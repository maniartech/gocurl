package main

// REF: https://chatgpt.com/c/670cdaf1-6854-800f-ab63-803af17c7110

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Constants for API endpoints
const (
	searchURL  = "https://api.nal.usda.gov/fdc/v1/foods/search"
	detailsURL = "https://api.nal.usda.gov/fdc/v1/food/%d"
)

// USDAClient represents a client to interact with the USDA API
type USDAClient struct {
	APIKey string
	Cache  *Cache
}

// Cache to store USDA data and reduce API calls
type Cache struct {
	FoodDetails map[int]Food
}

// Food represents a food item with its details
type Food struct {
	FdcID       int
	Description string
	Portions    []FoodPortion
}

// FoodPortion represents a portion size of a food item
type FoodPortion struct {
	MeasureUnit        string
	Modifier           string
	GramWeight         float64
	PortionDescription string
}

// Ingredient represents an ingredient for conversion
type Ingredient struct {
	Name     string
	FoodData Food
	Client   *USDAClient
}

// ConversionService provides methods for unit conversions
type ConversionService struct {
	Client *USDAClient
}

// NewUSDAClient creates a new USDAClient
func NewUSDAClient(apiKey string) *USDAClient {
	return &USDAClient{
		APIKey: apiKey,
		Cache: &Cache{
			FoodDetails: make(map[int]Food),
		},
	}
}

// SearchFood searches for a food item and returns its FdcID
func (client *USDAClient) SearchFood(query string) (int, error) {
	params := url.Values{}
	params.Add("api_key", client.APIKey)
	params.Add("query", query)
	params.Add("pageSize", "1")

	resp, err := http.Get(searchURL + "?" + params.Encode())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Foods []struct {
			Description string `json:"description"`
			FdcID       int    `json:"fdcId"`
		} `json:"foods"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if len(result.Foods) == 0 {
		return 0, fmt.Errorf("no foods found for query '%s'", query)
	}

	return result.Foods[0].FdcID, nil
}

// GetFoodDetails retrieves detailed information about a food item
func (client *USDAClient) GetFoodDetails(fdcID int) (Food, error) {
	// Check cache first
	if food, exists := client.Cache.FoodDetails[fdcID]; exists {
		return food, nil
	}

	params := url.Values{}
	params.Add("api_key", client.APIKey)

	url := fmt.Sprintf(detailsURL, fdcID) + "?" + params.Encode()
	resp, err := http.Get(url)
	if err != nil {
		return Food{}, err
	}
	defer resp.Body.Close()

	var details struct {
		Description  string `json:"description"`
		FoodPortions []struct {
			MeasureUnit struct {
				Name string `json:"name"`
			} `json:"measureUnit"`
			Modifier           string  `json:"modifier"`
			GramWeight         float64 `json:"gramWeight"`
			PortionDescription string  `json:"portionDescription"`
		} `json:"foodPortions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return Food{}, err
	}

	// Map FoodPortions
	portions := make([]FoodPortion, 0, len(details.FoodPortions))
	for _, p := range details.FoodPortions {
		portions = append(portions, FoodPortion{
			MeasureUnit:        p.MeasureUnit.Name,
			Modifier:           p.Modifier,
			GramWeight:         p.GramWeight,
			PortionDescription: p.PortionDescription,
		})
	}

	food := Food{
		FdcID:       fdcID,
		Description: details.Description,
		Portions:    portions,
	}

	// Store in cache
	client.Cache.FoodDetails[fdcID] = food

	return food, nil
}

// NewIngredient creates a new Ingredient
func NewIngredient(name string, client *USDAClient) (*Ingredient, error) {
	fdcID, err := client.SearchFood(name)
	if err != nil {
		return nil, err
	}

	foodData, err := client.GetFoodDetails(fdcID)
	if err != nil {
		return nil, err
	}

	return &Ingredient{
		Name:     name,
		FoodData: foodData,
		Client:   client,
	}, nil
}

// FindPortionByUnit finds a food portion matching the given unit
func (ingredient *Ingredient) FindPortionByUnit(unit string) (FoodPortion, error) {
	unit = normalizeUnit(unit)
	availableUnits := []string{}

	for _, portion := range ingredient.FoodData.Portions {
		unitName := normalizeUnit(portion.MeasureUnit)
		portionDesc := normalizeUnit(portion.PortionDescription)
		modifier := normalizeUnit(portion.Modifier)

		// Collect available units
		if unitName != "" {
			availableUnits = append(availableUnits, unitName)
		}
		if portionDesc != "" {
			availableUnits = append(availableUnits, portionDesc)
		}
		if modifier != "" {
			availableUnits = append(availableUnits, modifier)
		}

		// Matching logic
		if unitName == unit || portionDesc == unit || modifier == unit {
			return portion, nil
		}
	}

	availableUnits = removeDuplicates(availableUnits)
	return FoodPortion{}, fmt.Errorf("unit '%s' not found for ingredient '%s'. Available units: %v", unit, ingredient.Name, availableUnits)
}

// GetGramWeightPerUnit retrieves the gram weight per specified unit
func (ingredient *Ingredient) GetGramWeightPerUnit(unit string) (float64, error) {
	portion, err := ingredient.FindPortionByUnit(unit)
	if err != nil {
		return 0, err
	}
	return portion.GramWeight, nil
}

// ConvertVolumeToWeight converts volume to weight
func (service *ConversionService) ConvertVolumeToWeight(ingredient *Ingredient, unit string, amount float64) (float64, error) {
	gramWeightPerUnit, err := ingredient.GetGramWeightPerUnit(unit)
	if err != nil {
		return 0, err
	}
	totalWeight := amount * gramWeightPerUnit
	return totalWeight, nil
}

// ConvertWeightToVolume converts weight to volume
func (service *ConversionService) ConvertWeightToVolume(ingredient *Ingredient, unit string, weightInGrams float64) (float64, error) {
	gramWeightPerUnit, err := ingredient.GetGramWeightPerUnit(unit)
	if err != nil {
		return 0, err
	}
	amount := weightInGrams / gramWeightPerUnit
	return amount, nil
}

// GetAverageWeightBySize retrieves the average weight of an item by size
func (ingredient *Ingredient) GetAverageWeightBySize(size string) (float64, error) {
	size = normalizeUnit(size)
	availableSizes := []string{}

	for _, portion := range ingredient.FoodData.Portions {
		modifier := normalizeUnit(portion.Modifier)

		// Collect available sizes
		if modifier != "" {
			availableSizes = append(availableSizes, modifier)
		}

		// Matching logic
		if modifier == size || strings.Contains(modifier, size) {
			return portion.GramWeight, nil
		}
	}

	availableSizes = removeDuplicates(availableSizes)
	return 0, fmt.Errorf("size '%s' not found for '%s'. Available sizes: %v", size, ingredient.Name, availableSizes)
}

// ConvertWeightToItemCount converts weight to item count based on size
func (service *ConversionService) ConvertWeightToItemCount(ingredient *Ingredient, weightInGrams float64, size string) (float64, error) {
	avgWeight, err := ingredient.GetAverageWeightBySize(size)
	if err != nil {
		return 0, err
	}
	count := weightInGrams / avgWeight
	return count, nil
}

// ConvertItemCountToWeight converts item count to total weight based on size
func (service *ConversionService) ConvertItemCountToWeight(ingredient *Ingredient, count float64, size string) (float64, error) {
	avgWeight, err := ingredient.GetAverageWeightBySize(size)
	if err != nil {
		return 0, err
	}
	totalWeight := count * avgWeight
	return totalWeight, nil
}

func removeDuplicates(elements []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for _, v := range elements {
		v = strings.TrimSpace(v)
		if v != "" && !encountered[v] {
			encountered[v] = true
			result = append(result, v)
		}
	}
	return result
}

func normalizeUnit(unit string) string {
	unit = strings.ToLower(strings.TrimSpace(unit))
	unitMap := map[string]string{
		"cups":        "cup",
		"cup":         "cup",
		"tablespoons": "tablespoon",
		"tablespoon":  "tablespoon",
		"tbsp":        "tablespoon",
		"teaspoons":   "teaspoon",
		"teaspoon":    "teaspoon",
		"tsp":         "teaspoon",
		"grams":       "gram",
		"gram":        "gram",
		"g":           "gram",
		"kilograms":   "kilogram",
		"kilogram":    "kilogram",
		"kg":          "kilogram",
		"ounces":      "ounce",
		"ounce":       "ounce",
		"oz":          "ounce",
		"pounds":      "pound",
		"pound":       "pound",
		"lb":          "pound",
		"milliliters": "milliliter",
		"milliliter":  "milliliter",
		"ml":          "milliliter",
		"liters":      "liter",
		"liter":       "liter",
		"l":           "liter",
		// Add more units as needed
	}
	if normalized, exists := unitMap[unit]; exists {
		return normalized
	}
	return unit
}

func validateInputs(ingredientName, unit string, amount, weight float64) error {
	if ingredientName == "" {
		return errors.New("ingredient name is required")
	}
	if unit == "" {
		return errors.New("unit is required")
	}
	if amount == 0 && weight == 0 {
		return errors.New("either amount or weight must be provided")
	}
	return nil
}

func main() {
	// Load environment variables from .recipe.env file
	err := godotenv.Load(".recipe.env")
	if err != nil {
		log.Fatalf("Error loading .recipe.env file: %v", err)
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY not set in environment")
	}

	// Parse command-line flags
	ingredientName := flag.String("ingredient", "", "Name of the ingredient")
	unit := flag.String("unit", "", "Unit of measurement (e.g., cup, tablespoon)")
	amount := flag.Float64("amount", 0, "Amount in units (e.g., cups)")
	weight := flag.Float64("weight", 0, "Weight in grams")
	size := flag.String("size", "", "Size of the item (e.g., small, medium)")
	flag.Parse()

	// Validate inputs
	err = validateInputs(*ingredientName, *unit, *amount, *weight)
	if err != nil {
		log.Fatal(err)
	}

	client := NewUSDAClient(apiKey)
	conversionService := &ConversionService{Client: client}

	// Create Ingredient
	ingredient, err := NewIngredient(*ingredientName, client)
	if err != nil {
		log.Fatalf("Error creating ingredient: %v", err)
	}

	// If amount is provided, convert volume to weight
	if *amount > 0 {
		totalWeight, err := conversionService.ConvertVolumeToWeight(ingredient, *unit, *amount)
		if err != nil {
			fmt.Println("Error during volume to weight conversion:", err)
		} else {
			fmt.Printf("\n%.2f %s of %s equals %.2f grams\n", *amount, *unit, ingredient.FoodData.Description, totalWeight)
		}
	}

	// If weight is provided, convert weight to volume
	if *weight > 0 {
		totalVolume, err := conversionService.ConvertWeightToVolume(ingredient, *unit, *weight)
		if err != nil {
			fmt.Println("Error during weight to volume conversion:", err)
		} else {
			fmt.Printf("%.2f grams of %s equals %.2f %s\n", *weight, ingredient.FoodData.Description, totalVolume, *unit)
		}
	}

	// If size is provided, perform count conversions
	if *size != "" && *weight > 0 {
		// Weight to Item Count
		itemCount, err := conversionService.ConvertWeightToItemCount(ingredient, *weight, *size)
		if err != nil {
			fmt.Println("Error converting weight to item count:", err)
		} else {
			fmt.Printf("%.2f grams equals approximately %.2f %s %s\n", *weight, itemCount, *size, ingredient.FoodData.Description)
		}

		// Item Count to Weight
		totalItemWeight, err := conversionService.ConvertItemCountToWeight(ingredient, itemCount, *size)
		if err != nil {
			fmt.Println("Error converting item count to weight:", err)
		} else {
			fmt.Printf("%.2f %s %s weigh approximately %.2f grams\n", itemCount, *size, ingredient.FoodData.Description, totalItemWeight)
		}
	}
}
