# Exercise 3: CI/CD Integration

**Difficulty:** Advanced
**Duration:** 60-90 minutes
**Prerequisites:** CI/CD platform access, Exercises 1-2 completed

## Objective

Integrate gocurl into CI/CD pipelines for automated API testing, deployment validation, and health monitoring. Learn to create robust automation workflows across GitHub Actions, GitLab CI, and Jenkins.

## Tasks

### Task 1: GitHub Actions Workflow

Create `.github/workflows/api-tests.yml` that runs API tests on every push.

**Requirements:**
- Trigger on push and pull_request
- Install gocurl
- Run comprehensive API tests
- Fail build if tests fail
- Upload test results as artifacts

**Starter Template:**
```yaml
name: API Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  api-tests:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install GoCurl
        run: |
          # TODO: Install gocurl CLI

      - name: Run API Tests
        env:
          API_URL: ${{ secrets.API_URL }}
          API_KEY: ${{ secrets.API_KEY }}
        run: |
          # TODO: Run your API tests

      - name: Upload Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: # TODO: Specify results path
```

**Complete this workflow to:**
1. Install gocurl
2. Run at least 10 API tests
3. Generate test report (JSON or text)
4. Upload results

<details>
<summary>💡 Hint</summary>

```bash
go install github.com/maniartech/gocurl/cmd/gocurl@latest
echo "$HOME/go/bin" >> $GITHUB_PATH
```
</details>

---

### Task 2: GitLab CI Pipeline

Create `.gitlab-ci.yml` with multiple stages for API validation.

**Requirements:**
- Stages: test, validate, deploy
- Test stage: Run API tests
- Validate stage: Check deployment health
- Deploy stage: Deploy only if tests pass
- Use environment variables
- Cache gocurl installation

**Starter Template:**
```yaml
stages:
  - test
  - validate
  - deploy

variables:
  API_URL: "https://api.staging.example.com"

# TODO: Add cache configuration

api-tests:
  stage: test
  image: golang:1.21
  script:
    # TODO: Install gocurl
    # TODO: Run tests
    # TODO: Generate report
  artifacts:
    reports:
      junit: # TODO: Add report path
    paths:
      - test-results/

health-check:
  stage: validate
  image: golang:1.21
  script:
    # TODO: Check API health
    # TODO: Verify endpoints
  only:
    - main
    - develop

deploy:
  stage: deploy
  script:
    # TODO: Deployment steps
    # TODO: Post-deployment validation
  only:
    - main
```

**Complete this pipeline:**
1. Cache gocurl binary
2. Create comprehensive test script
3. Add health check validation
4. Implement deployment verification

---

### Task 3: Jenkins Pipeline

Create `Jenkinsfile` for API testing and monitoring.

**Requirements:**
- Declarative pipeline syntax
- Parallel test execution
- Slack/email notifications on failure
- Publish test results
- Archive artifacts

**Starter Template:**
```groovy
pipeline {
    agent any

    environment {
        API_URL = credentials('api-url')
        API_KEY = credentials('api-key')
    }

    stages {
        stage('Setup') {
            steps {
                script {
                    // TODO: Install gocurl
                }
            }
        }

        stage('API Tests') {
            parallel {
                stage('Smoke Tests') {
                    steps {
                        // TODO: Run smoke tests
                    }
                }
                stage('Integration Tests') {
                    steps {
                        // TODO: Run integration tests
                    }
                }
                stage('Performance Tests') {
                    steps {
                        // TODO: Run performance tests
                    }
                }
            }
        }

        stage('Report') {
            steps {
                // TODO: Generate and publish reports
            }
        }
    }

    post {
        failure {
            // TODO: Send notifications
        }
        always {
            // TODO: Archive artifacts
        }
    }
}
```

**Complete this pipeline:**
1. Add gocurl installation
2. Create test scripts for each stage
3. Add notification logic
4. Publish JUnit test results

---

### Task 4: Pre-deployment Validation Script

Create `pre-deploy.sh` that validates deployment readiness.

**Requirements:**
- Check all critical endpoints
- Verify database connectivity
- Check external service dependencies
- Validate environment configuration
- Exit with error if any check fails

**Starter Code:**
```bash
#!/bin/bash
set -e

API_URL="${API_URL:-https://api.staging.example.com}"
CHECKS_PASSED=0
CHECKS_FAILED=0

echo "Pre-Deployment Validation"
echo "=========================="
echo "Environment: ${ENV:-staging}"
echo "API URL: $API_URL"
echo ""

# TODO: Implement check functions

check_endpoint() {
    local name=$1
    local endpoint=$2
    local expected_status=${3:-200}

    # TODO: Implement endpoint check
}

check_database() {
    # TODO: Check database health endpoint
}

check_external_services() {
    # TODO: Check external dependencies
}

# Run all checks
check_endpoint "Health Check" "/health" 200
check_endpoint "API Status" "/api/v1/status" 200
check_database
check_external_services

# TODO: Print summary and exit
echo ""
echo "Results: $CHECKS_PASSED passed, $CHECKS_FAILED failed"
if [ $CHECKS_FAILED -gt 0 ]; then
    echo "❌ Pre-deployment validation failed!"
    exit 1
else
    echo "✅ All checks passed! Ready to deploy."
    exit 0
fi
```

**Complete this script:**
1. Implement all check functions
2. Add timeout handling
3. Add retry logic for flaky checks
4. Generate detailed failure reports

---

### Task 5: Post-deployment Smoke Tests

Create `smoke-tests.sh` that runs immediately after deployment.

**Requirements:**
- Test all critical user journeys
- Verify new features work
- Check for regressions
- Run in under 60 seconds
- Integrate with rollback logic

**Starter Code:**
```bash
#!/bin/bash
API_URL="${1:-https://api.production.example.com}"
FAILURES=0

echo "🔥 Post-Deployment Smoke Tests"
echo "================================"
echo "Target: $API_URL"
echo "Started: $(date)"
echo ""

# Test 1: User Registration
echo "Test 1: User Registration"
response=$(gocurl -X POST \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"Test123!"}' \
    -w "%{http_code}" \
    -s \
    "$API_URL/api/v1/register")

# TODO: Parse and validate response

# TODO: Add more smoke tests
# - User login
# - Data retrieval
# - Search functionality
# - Payment processing (if applicable)

# TODO: Print summary
echo ""
echo "Smoke Tests Complete"
echo "===================="
echo "Total Failures: $FAILURES"

if [ $FAILURES -gt 0 ]; then
    echo "❌ SMOKE TESTS FAILED - INITIATE ROLLBACK"
    exit 1
else
    echo "✅ All smoke tests passed"
    exit 0
fi
```

**Complete this script:**
1. Add 10+ critical smoke tests
2. Implement response validation
3. Add timing constraints
4. Create rollback trigger logic

---

### Task 6: API Load Testing Script

Create `load-test.sh` for simple load testing.

**Requirements:**
- Concurrent requests (50-100)
- Measure response times
- Calculate success rate
- Generate performance report
- Detect anomalies

**Starter Code:**
```bash
#!/bin/bash
URL="${1:-https://httpbin.org/get}"
CONCURRENT=${2:-50}
TOTAL=${3:-500}

echo "Load Testing: $URL"
echo "Concurrent: $CONCURRENT | Total: $TOTAL"
echo "========================================"

start_time=$(date +%s)
success_count=0
failure_count=0

# TODO: Implement concurrent request logic

# TODO: Calculate statistics

end_time=$(date +%s)
duration=$((end_time - start_time))

echo ""
echo "Load Test Complete"
echo "=================="
echo "Duration: ${duration}s"
echo "Success: $success_count"
echo "Failure: $failure_count"
echo "Requests/sec: $((TOTAL / duration))"
```

**Complete this script:**
1. Make concurrent requests
2. Collect response times
3. Calculate percentiles (p50, p95, p99)
4. Generate detailed report

---

## Validation

For this exercise, validation is deployment-specific. Here's a checklist:

### GitHub Actions Validation
- [ ] Workflow triggers on push
- [ ] GoCurl installs successfully
- [ ] Tests execute and report results
- [ ] Artifacts upload correctly
- [ ] Build fails on test failure

### GitLab CI Validation
- [ ] Pipeline has 3 stages
- [ ] Cache works correctly
- [ ] Parallel jobs execute
- [ ] Artifacts persist between stages
- [ ] Only runs on specified branches

### Jenkins Validation
- [ ] Pipeline parses successfully
- [ ] Parallel stages execute
- [ ] Credentials inject correctly
- [ ] Test results publish
- [ ] Notifications trigger on failure

### Scripts Validation
- [ ] Pre-deployment script catches issues
- [ ] Smoke tests run in <60s
- [ ] Load test generates accurate stats
- [ ] All scripts exit with correct codes
- [ ] Error messages are clear

## Integration Test

Create an end-to-end test:

```bash
#!/bin/bash
# integration-test.sh

echo "Running Full CI/CD Integration Test"
echo "===================================="

# Step 1: Pre-deployment validation
echo "Step 1: Pre-deployment validation"
./pre-deploy.sh || { echo "❌ Pre-deploy failed"; exit 1; }

# Step 2: Deploy (simulated)
echo "Step 2: Deploying... (simulated)"
sleep 2
echo "✅ Deployment complete"

# Step 3: Smoke tests
echo "Step 3: Running smoke tests"
./smoke-tests.sh || { echo "❌ Smoke tests failed, rolling back..."; exit 1; }

# Step 4: Load testing
echo "Step 4: Running load tests"
./load-test.sh https://httpbin.org/get 20 100

echo ""
echo "✅ Full CI/CD integration test passed!"
```

## Bonus Challenges

1. **Blue-Green Deployment:** Create scripts for blue-green deployment with automatic rollback
2. **Canary Deployment:** Implement canary deployment with progressive traffic shifting
3. **Performance Baseline:** Create system to track performance trends over time
4. **Auto-scaling Trigger:** Script that triggers scaling based on API response times
5. **Multi-region Validation:** Validate deployment across multiple regions simultaneously

## Learning Outcomes

After completing this exercise, you should be able to:
- ✅ Integrate gocurl into GitHub Actions workflows
- ✅ Create GitLab CI pipelines with multiple stages
- ✅ Build Jenkins declarative pipelines
- ✅ Implement pre-deployment validation
- ✅ Create post-deployment smoke tests
- ✅ Perform basic load testing
- ✅ Automate API testing in CI/CD
- ✅ Implement deployment safety checks

## Real-World Application

These skills apply directly to:
- **Continuous Testing:** Automated API validation on every commit
- **Deployment Safety:** Prevent bad deployments with validation gates
- **Production Monitoring:** Continuous health checks post-deployment
- **Performance Tracking:** Monitor API performance over time
- **Incident Response:** Quick validation during incidents

## Next Steps

1. Implement these workflows in a real project
2. Add monitoring and alerting
3. Create dashboards for test results
4. Document runbooks for common failures
5. Move on to Part II: API Approaches (if you've completed all Part I exercises)

---

**Congratulations!** You've completed all Chapter 4 exercises. You now have practical experience with the GoCurl CLI in development, scripting, and production environments.
