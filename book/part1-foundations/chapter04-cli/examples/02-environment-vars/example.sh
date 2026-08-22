#!/bin/bash
# Example 2: Environment Variables
# Demonstrates using environment variables in gocurl commands

echo "🔐 Example 2: Environment Variables"
echo "===================================="
echo ""

# Set environment variables
export API_URL="https://httpbin.org"
export API_TOKEN="secret-token-12345"
export USER_NAME="Alice"

echo "📝 Environment variables set:"
echo "   API_URL=$API_URL"
echo "   API_TOKEN=$API_TOKEN"
echo "   USER_NAME=$USER_NAME"
echo ""

# Example 1: Use environment variable in URL
echo "1️⃣  Using Environment Variable in URL"
echo "Command: gocurl \$API_URL/get"
gocurl $API_URL/get
echo ""

# Example 2: Use environment variable in header
echo "2️⃣  Using Environment Variable in Header"
echo "Command: gocurl -H 'Authorization: Bearer \$API_TOKEN' \$API_URL/headers"
gocurl -H "Authorization: Bearer $API_TOKEN" $API_URL/headers
echo ""

# Example 3: Use multiple environment variables
echo "3️⃣  Using Multiple Environment Variables"
echo "Command: gocurl \$API_URL/get?user=\$USER_NAME"
gocurl $API_URL/get?user=$USER_NAME
echo ""

# Example 4: Braces syntax
echo "4️⃣  Using Braces Syntax \${VAR}"
echo "Command: gocurl \${API_URL}/anything?token=\${API_TOKEN}"
gocurl ${API_URL}/anything?token=${API_TOKEN}
echo ""

# Example 5: Configuration file pattern
echo "5️⃣  Configuration File Pattern"
echo "Creating .env.example..."

cat > .env.example << 'EOF'
# API Configuration
API_URL=https://api.example.com
API_KEY=your-api-key-here
API_VERSION=v1
TIMEOUT=30
EOF

echo "Contents of .env.example:"
cat .env.example
echo ""
echo "Usage: source .env.example && gocurl \$API_URL/\$API_VERSION/users"
echo ""

# Cleanup
rm .env.example

# Example 6: Secure secrets pattern
echo "6️⃣  Secure Secrets Pattern"
echo "Command: export SECRET=\$(cat secret.txt) && gocurl -H 'X-Secret: \$SECRET' \$API_URL/headers"
echo "secret-from-file" > secret.txt
export SECRET=$(cat secret.txt)
gocurl -H "X-Secret: $SECRET" $API_URL/headers
rm secret.txt
echo ""

echo "✅ Environment variables examples complete!"
echo ""
echo "💡 Best Practices:"
echo "   • Store secrets in environment, not in code"
echo "   • Use .env files for configuration"
echo "   • Never commit secrets to version control"
echo "   • Use \${VAR} syntax for clarity"
