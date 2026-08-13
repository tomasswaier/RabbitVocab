{
  description = "Go flake";
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
        devShell = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.gopls
            pkgs.golangci-lint
						pkgs.sqlc
          ];
          
        };
				shellHook = ''
				  export PATH="$PATH:$(go env GOPATH)/bin"
				  echo "MCP development shell ready!"
				  echo "Run: mcp-inspector ./server/server"
				'';
      }
    );
}
