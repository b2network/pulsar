# Contributing to Pulsar

Thank you for considering contributing to Pulsar!

## Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR-USERNAME/pulsar.git
   cd pulsar
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/b2network/pulsar.git
   ```
4. Install dependencies:
   ```bash
   make deps
   ```

## Development Workflow

1. Create a new branch for your feature/fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and ensure they follow the coding standards

3. Run tests:
   ```bash
   make test
   ```

4. Run linters:
   ```bash
   golangci-lint run
   ```

5. Build the project:
   ```bash
   make build
   ```

6. Commit your changes following conventional commits:
   ```bash
   git commit -m "feat: add new feature"
   ```

## Code Style

- Follow Go best practices and conventions
- Use `gofmt` and `goimports` for formatting
- Add comments for exported functions and types
- Keep functions small and focused
- Write tests for new features

## Commit Message Format

We use conventional commits format:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `style:` Code style changes (formatting, etc)
- `refactor:` Code refactoring
- `test:` Test changes
- `chore:` Build process or auxiliary tool changes

## Pull Request Process

1. Update your branch with the latest upstream changes:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. Push your branch to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

3. Create a Pull Request from your fork to the main repository

4. Ensure all CI checks pass

5. Wait for code review and address any feedback

## Testing

- Write unit tests for new functionality
- Ensure all existing tests pass
- Add integration tests for complex features
- Test coverage should not decrease

## Documentation

- Update README.md if needed
- Add inline documentation for complex logic
- Update API documentation for new endpoints
- Add examples for new features

## Questions?

Feel free to open an issue for any questions or discussions.