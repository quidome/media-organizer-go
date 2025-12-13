{
  description = "Media Organizer Go - Development Environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go
            gopls
            gotools
            go-tools
            
            # Code quality tools
            golangci-lint
            golines
            goimports-reviser
            
            # Development utilities
            delve
            air
            
            # Additional tools
            git
          ];

          shellHook = ''
            echo "🚀 Media Organizer Go Development Environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Available commands:"
            echo "  go build ./...           - Build the project"
            echo "  go test ./...            - Run all tests"
            echo "  golangci-lint run        - Lint the code"
            echo "  gofmt -s -w .            - Format code"
            echo "  dlv debug                - Debug with Delve"
            echo ""
          '';

          # Set Go environment variables
          GOROOT = "${pkgs.go}/share/go";
        };
      }
    );
}
