#!/bin/bash
# Example 5: Multi-Environment Configuration
# Demonstrates switching between dev/staging/prod environments

echo "🌍 Example 5: Multi-Environment Configuration"
echo "=============================================="
echo ""

# Create environment configurations
echo "📝 Creating environment configurations..."
echo ""

# Dev environment
cat > .env.dev << 'EOF'
export ENV_NAME="Development"
export API_URL="https://api.dev.example.com"
export API_KEY="dev-key-12345"
export TIMEOUT="10"
export LOG_LEVEL="debug"
EOF

# Staging environment
cat > .env.staging << 'EOF'
export ENV_NAME="Staging"
export API_URL="https://api.staging.example.com"
export API_KEY="staging-key-67890"
export TIMEOUT="30"
export LOG_LEVEL="info"
EOF

# Production environment
cat > .env.prod << 'EOF'
export ENV_NAME="Production"
export API_URL="https://api.prod.example.com"
export API_KEY="prod-key-abcdef"
export TIMEOUT="60"
export LOG_LEVEL="warn"
EOF

# Example 1: Load development environment
echo "1️⃣  Load Development Environment"
source .env.dev
echo "Environment: $ENV_NAME"
echo "API URL: $API_URL"
echo "API Key: ${API_KEY:0:8}..."
echo "Timeout: ${TIMEOUT}s"
echo ""
echo "Command: gocurl -H 'X-API-Key: \$API_KEY' \$API_URL/health"
echo "(Using httpbin.org for demo since dev.example.com doesn't exist)"
gocurl -H "X-API-Key: $API_KEY" -H "X-Environment: $ENV_NAME" https://httpbin.org/headers
echo ""

# Example 2: Load staging environment
echo "2️⃣  Load Staging Environment"
source .env.staging
echo "Environment: $ENV_NAME"
echo "API URL: $API_URL"
echo "API Key: ${API_KEY:0:8}..."
echo ""
gocurl -H "X-API-Key: $API_KEY" -H "X-Environment: $ENV_NAME" https://httpbin.org/headers
echo ""

# Example 3: Load production environment
echo "3️⃣  Load Production Environment"
source .env.prod
echo "Environment: $ENV_NAME"
echo "API URL: $API_URL"
echo "API Key: ${API_KEY:0:8}..."
echo ""
gocurl -H "X-API-Key: $API_KEY" -H "X-Environment: $ENV_NAME" https://httpbin.org/headers
echo ""

# Example 4: Environment switcher script
echo "4️⃣  Environment Switcher Script"
cat > env-switch.sh << 'EOF'
#!/bin/bash
ENV=${1:-dev}

case $ENV in
    dev|development)
        source .env.dev
        ;;
    staging|stage)
        source .env.staging
        ;;
    prod|production)
        source .env.prod
        ;;
    *)
        echo "Unknown environment: $ENV"
        echo "Usage: $0 [dev|staging|prod]"
        exit 1
        ;;
esac

echo "✅ Loaded $ENV_NAME environment"
echo "   URL: $API_URL"
echo "   Timeout: ${TIMEOUT}s"
EOF

chmod +x env-switch.sh
echo "Running: source ./env-switch.sh dev"
source ./env-switch.sh dev
echo ""

# Example 5: Environment-aware API call script
echo "5️⃣  Environment-Aware API Call Script"
cat > api-call.sh << 'EOF'
#!/bin/bash
set -e

ENV=${1:-dev}
ENDPOINT=${2:-/health}

# Load environment
case $ENV in
    dev) source .env.dev ;;
    staging) source .env.staging ;;
    prod) source .env.prod ;;
    *)
        echo "Unknown environment: $ENV"
        exit 1
        ;;
esac

echo "📍 Calling $ENV_NAME API"
echo "   Endpoint: $ENDPOINT"
echo "   URL: $API_URL$ENDPOINT"
echo ""

# Make request with environment-specific config
gocurl \
    -H "X-API-Key: $API_KEY" \
    -H "X-Environment: $ENV_NAME" \
    --max-time "$TIMEOUT" \
    "${API_URL}${ENDPOINT}" || {
        echo "❌ Request failed"
        exit 1
    }

echo ""
echo "✅ Request successful"
EOF

chmod +x api-call.sh
echo "Created api-call.sh"
echo "Usage: ./api-call.sh [dev|staging|prod] [/endpoint]"
echo ""

# Example 6: Compare environments
echo "6️⃣  Compare All Environments"
cat > compare-envs.sh << 'EOF'
#!/bin/bash
echo "Comparing all environments:"
echo ""

for env_file in .env.dev .env.staging .env.prod; do
    source "$env_file"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "$ENV_NAME"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  URL:      $API_URL"
    echo "  Key:      ${API_KEY:0:12}..."
    echo "  Timeout:  ${TIMEOUT}s"
    echo "  Log Level: $LOG_LEVEL"
    echo ""
done
EOF

chmod +x compare-envs.sh
./compare-envs.sh

# Cleanup
rm -f .env.dev .env.staging .env.prod env-switch.sh api-call.sh compare-envs.sh

echo "✅ Multi-environment examples complete!"
echo ""
echo "💡 Best Practices:"
echo "   • Keep separate .env files per environment"
echo "   • Never commit .env files to git (add to .gitignore)"
echo "   • Use environment switcher scripts"
echo "   • Validate environment before making requests"
echo "   • Use different API keys per environment"
echo "   • Set appropriate timeouts per environment"
