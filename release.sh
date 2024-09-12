# This tags the current commit with the next version based on conventional commits and pushes the tag.
# This triggers a GitHub Action that builds and releases the application.
go install github.com/caarlos0/svu@latest
git tag $(svu next)
git push --tags