#!/bin/bash

# Demo project setup script for testing GitHub Copilot with go-zero

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

PROJECT_NAME="userapidemo"
DEMO_DIR="$(pwd)/demo-workspace"

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}go-zero AI Ecosystem Demo Project Setup${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Check dependencies
echo "Checking dependencies..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed. Please install Go first.${NC}"
    exit 1
fi

if ! command -v goctl &> /dev/null; then
    echo -e "${YELLOW}Warning: goctl not installed, installing...${NC}"
    go install github.com/zeromicro/go-zero/tools/goctl@latest
fi

echo -e "${GREEN}✓ Dependencies check completed${NC}"
echo ""

# Create demo directory
echo "Creating demo project directory: $DEMO_DIR"
mkdir -p "$DEMO_DIR"
cd "$DEMO_DIR"

# Initialize git repository
if [ ! -d ".git" ]; then
    git init -q
    echo -e "${GREEN}✓ Git repository initialized${NC}"
fi

# Configure GitHub Copilot (add ai-context)
echo ""
echo "Configuring GitHub Copilot..."
if [ ! -d ".github/ai-context" ]; then
    git submodule add -q https://github.com/zeromicro/ai-context.git .github/ai-context 2>/dev/null || echo "Submodule already exists"
    mkdir -p .github
    ln -sf ai-context/00-instructions.md .github/copilot-instructions.md
    echo -e "${GREEN}✓ GitHub Copilot configured${NC}"
    echo -e "  - Submodule: .github/ai-context"
    echo -e "  - Instructions: .github/copilot-instructions.md"
else
    echo -e "${YELLOW}⚠ GitHub Copilot already configured${NC}"
fi

# Create project structure
echo ""
echo "Creating go-zero API project structure..."
if [ -d "$PROJECT_NAME" ]; then
    echo -e "${YELLOW}⚠ Project exists, removing and recreating...${NC}"
    rm -rf "$PROJECT_NAME"
fi

mkdir -p "$PROJECT_NAME"
cd "$PROJECT_NAME"

# Create API definition file
echo ""
echo "Creating API definition file..."
cat > user.api << 'EOF'
syntax = "v1"

info (
	title:   "User Service API"
	desc:    "User management API"
	author:  "go-zero"
	version: "1.0"
)

type (
	// User information
	User {
		Id       int64  `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		CreateAt string `json:"create_at"`
	}

	// Create user request
	CreateUserRequest {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Create user response
	CreateUserResponse {
		User User `json:"user"`
	}

	// Get user request
	GetUserRequest {
		Id int64 `path:"id"`
	}

	// Get user response
	GetUserResponse {
		User User `json:"user"`
	}

	// List users request
	ListUsersRequest {
		Page     int `form:"page,default=1"`
		PageSize int `form:"page_size,default=10"`
	}

	// List users response
	ListUsersResponse {
		Users []User `json:"users"`
		Total int64  `json:"total"`
	}
)

service user-api {
	@doc "Create user"
	@handler CreateUser
	post /api/users (CreateUserRequest) returns (CreateUserResponse)

	@doc "Get user details"
	@handler GetUser
	get /api/users/:id (GetUserRequest) returns (GetUserResponse)

	@doc "List users"
	@handler ListUsers
	get /api/users (ListUsersRequest) returns (ListUsersResponse)
}
EOF

echo -e "${GREEN}✓ API definition file created: user.api${NC}"

# Generate code
echo ""
echo "Generating go-zero code..."
goctl api go -api user.api -dir . -style go_zero

echo -e "${GREEN}✓ Code generation completed${NC}"

# Initialize go module
echo ""
echo "Initializing Go module..."
if [ ! -f "go.mod" ]; then
    go mod init $PROJECT_NAME
fi
go mod tidy

echo -e "${GREEN}✓ Go module initialized${NC}"

# Create .gitignore
cat > .gitignore << 'EOF'
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
userapidemo

# Go workspace
go.work
go.work.sum

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db
EOF

echo -e "${GREEN}✓ .gitignore created${NC}"

# Build project to verify
echo ""
echo "Building project to verify..."
if go build -o userapidemo .; then
    echo -e "${GREEN}✓ Build successful${NC}"
    rm -f userapidemo  # Clean up binary
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi

# Create README
cat > README.md << 'EOF'
# User API Demo - go-zero with GitHub Copilot

This is a go-zero demo project configured with GitHub Copilot + ai-context.

## Project Structure

```
userapidemo/
├── etc/              # Configuration files
├── internal/
│   ├── handler/      # HTTP handlers
│   ├── logic/        # Business logic
│   ├── svc/          # Service context
│   └── types/        # Type definitions
├── user.api          # API definition file
└── userapidemo.go    # Main entry point
```

## Running the Service

```bash
# Run the service
go run userapidemo.go -f etc/user-api.yaml

# The service will start on http://localhost:8888
```

## Testing with curl

```bash
# Create a user
curl -X POST http://localhost:8888/api/users \
  -H "Content-Type: application/json" \
  -d '{"username":"john","email":"john@example.com","password":"secret"}'

# Get user by ID
curl http://localhost:8888/api/users/1

# List users
curl http://localhost:8888/api/users?page=1&page_size=10
```

## Testing GitHub Copilot

### Scenario 1: Implement CreateUser Logic

1. Open `internal/logic/createuserlogic.go` in VS Code
2. In the `CreateUser` method, try typing:
```go
// TODO: Validate username is not empty
```

3. **Expected behavior**:
   - ✅ Copilot suggests go-zero error handling
   - ✅ Uses proper error returns
   - ✅ Follows Logic layer responsibilities
   - ✅ Correctly uses `req` and `types` definitions

### Scenario 2: Add Database Model

1. Try creating a comment in the project:
```go
// TODO: Add MySQL user table model
```

2. **Expected behavior**:
   - Copilot suggests using `goctl model` commands
   - Suggests proper model structure
   - Includes database connection in ServiceContext

### Scenario 3: Add Middleware

1. Create `internal/middleware/auth.go`
2. Try typing:
```go
// TODO: JWT authentication middleware
```

3. **Expected behavior**:
   - Copilot suggests middleware pattern
   - Includes proper context handling
   - Shows how to register in routes

## Implementation Tips

The business logic should be implemented in the `internal/logic/` directory:

- **createuserlogic.go**: Validate input, create user, return response
- **getuserlogic.go**: Fetch user by ID, handle not found
- **listuserslogic.go**: Paginate users, return list with total count

Use the Handler → Logic → Model three-layer architecture.

## Resources

- [go-zero Official Docs](https://go-zero.dev)
- [zero-skills Pattern Guides](https://github.com/zeromicro/zero-skills)
- [ai-context Instructions](https://github.com/zeromicro/ai-context)
EOF

echo -e "${GREEN}✓ README.md created${NC}"

# Return to demo-workspace root
cd "$DEMO_DIR"

# Create verification script
cat > verify-copilot.sh << 'EOF'
#!/bin/bash

# Verification script for GitHub Copilot configuration

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Verifying GitHub Copilot configuration..."
echo ""

# Check 1: ai-context submodule exists
if [ -d ".github/ai-context" ]; then
    echo -e "${GREEN}✓${NC} ai-context submodule exists"
else
    echo -e "${RED}✗${NC} ai-context submodule does not exist"
    exit 1
fi

# Check 2: copilot-instructions.md symlink exists
if [ -L ".github/copilot-instructions.md" ]; then
    echo -e "${GREEN}✓${NC} copilot-instructions.md symlink exists"
else
    echo -e "${RED}✗${NC} copilot-instructions.md symlink does not exist"
    exit 1
fi

# Check 3: Configuration file contains go-zero content
if grep -q "go-zero" .github/copilot-instructions.md; then
    echo -e "${GREEN}✓${NC} Configuration file contains go-zero content"
else
    echo -e "${RED}✗${NC} Configuration file does not contain go-zero content"
    exit 1
fi

# Check 4: Project structure is correct
if [ -d "userapidemo/internal" ]; then
    echo -e "${GREEN}✓${NC} go-zero project structure is correct"
else
    echo -e "${RED}✗${NC} go-zero project structure is incorrect"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ All checks passed!${NC}"
echo ""
echo "You can now open the project in VS Code and GitHub Copilot will use go-zero context!"
echo ""
echo "  cd $(pwd)/userapidemo"
echo "  code ."
EOF

chmod +x verify-copilot.sh

echo ""
echo -e "${BLUE}================================================${NC}"
echo -e "${GREEN}✓ Demo project setup completed!${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""
echo "Project location: $DEMO_DIR/$PROJECT_NAME"
echo ""
echo "Next steps:"
echo ""
echo "1. Verify configuration:"
echo -e "   ${YELLOW}cd $DEMO_DIR && ./verify-copilot.sh${NC}"
echo ""
echo "2. Open project in VS Code:"
echo -e "   ${YELLOW}cd $DEMO_DIR/$PROJECT_NAME${NC}"
echo -e "   ${YELLOW}code .${NC}"
echo ""
echo "3. Test GitHub Copilot:"
echo "   - Open internal/logic/createuserlogic.go"
echo "   - Try implementing the CreateUser method"
echo "   - Copilot will provide go-zero compliant suggestions based on ai-context"
echo ""
echo "4. Run the service:"
echo -e "   ${YELLOW}go run userapidemo.go -f etc/user-api.yaml${NC}"
echo ""
