# CLAUDE.md - alignment-research-feed (Go API)

## Validation

To validate changes in this repository, run the following automated checks:

- Linting: make lint
- OpenAPI spec validation: make lint-openapi
- Tests: make test-short

All automated steps must pass before changes can be merged.

Additionally, perform the following manual checks:

### Manual Quality Checks

After automated validation passes, review code for these issues:

1. **Unnecessarily Optional Fields**: Check for dependencies being defensively allowed to be optional, and simplify by assuming we always provide them.

2. **Duplicate Code**: Check for repeated definitions (constants, types, helper functions) that should be consolidated.

3. **Incomplete or Disconnected Logic**: Look for:
   - Legacy mechanisms or functions left in the code instead of being removed
   - Closures closing over data that may be outdated
   - Fields in types/interfaces that are never read or written
   - Functions that are defined but never called
   - Features partially implemented but not wired up
   - TODO comments indicating unfinished work
   - Missing expected env vars in .env.dist

4. **Validation Script Coverage**: Ensure any new validations are:
   - Added to CI

5. **OpenAPI Spec Completeness**: When modifying API endpoints, verify that `openapi/api.yaml`:
   - Includes all API endpoints from `internal/transport/web/router/router.go`
   - Has correct request/response schemas matching the controller implementations
   - Documents all query parameters, path parameters, and request bodies
   - Lists all possible HTTP status codes for each endpoint

6. **MCP Client Parity**: When modifying the domain `Article` struct or API endpoints, verify that the MCP client (`cmd/mcp/client/client.go`) stays in sync:
   - `client.Article` fields match `domain.Article` fields
   - Any new API endpoints are exposed as MCP tools in `cmd/mcp/server/`

7. **Redundant fields or parameters**: Look for:
   - Fields that contain information present in or inferrable from other fields
   - Parameters that contain information present in or inferrable from other fields

8. **Poor use of types**: Look for:
   - String comparisons on errors instead of using typed errors and errors.Is or errors.As

All steps must pass before changes can be merged.