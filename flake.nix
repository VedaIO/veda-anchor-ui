{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };
  outputs = {
    self,
    nixpkgs,
  }: let
    system = "x86_64-linux"; # or "aarch64-darwin" for M1/M2 Macs
    pkgs = import nixpkgs {inherit system;};
  in {
    devShells.${system}.default = pkgs.mkShell {
      buildInputs = with pkgs; [
        go_1_26
        golangci-lint
        gnumake
        bun
        biome
      ];
      shellHook = ''
        export GOPATH="$HOME/.local/share/go"
        export GOBIN="$GOPATH/bin"
        export PATH="$GOBIN:$PATH"
        export PATH="$HOME/.local/bin:$PATH"

        export BUN_INSTALL="$HOME/.local/share/bun"
        export PATH="$BUN_INSTALL/bin:$PATH"
        export BUN_INSTALL_CACHE_DIR="$HOME/.cache/bun"
      '';
    };
  };
}
