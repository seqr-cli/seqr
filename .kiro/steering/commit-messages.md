---
inclusion: always
---

# Commit Message Guidelines

When completing any task or implementing functionality, always write a clear and descriptive commit message following these guidelines:

## Format
```
<type>: <short description>

<detailed description>
- List key changes made
- Include any breaking changes
- Reference task/issue numbers if applicable
```

## Types
- `feat`: New feature implementation
- `fix`: Bug fixes
- `refactor`: Code refactoring without functional changes
- `test`: Adding or updating tests
- `docs`: Documentation updates
- `chore`: Maintenance tasks, build changes, etc.

## Requirements
- Always write commit messages when tasks are completed
- Include task references (e.g., "Closes task 1.2")
- List key implementation details
- Mention any breaking changes
- Keep the first line under 72 characters
- Use present tense ("add" not "added")
- Be specific about what was implemented

## Example
```
feat: implement user authentication system

- Add JWT-based authentication middleware
- Implement login/logout endpoints
- Add password hashing with bcrypt
- Create user session management
- Add authentication tests

Closes task AUTH-001
```