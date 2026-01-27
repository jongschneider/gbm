{
  description = "storybook-go development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    pre-commit-hooks.url = "github:cachix/pre-commit-hooks.nix";
  };

  outputs = {
    self,
    nixpkgs,
    pre-commit-hooks,
  }: let
    systems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];
    forAllSystems = nixpkgs.lib.genAttrs systems;
  in {
    checks = forAllSystems (system: {
      pre-commit-check = pre-commit-hooks.lib.${system}.run {
        src = ./.;
        hooks = {
          golangci-lint-fmt = {
            enable = true;
            name = "golangci-lint fmt";
            entry = "golangci-lint fmt";
            types = ["go"];
            pass_filenames = false;
          };
        };
      };
    });

    devShells = forAllSystems (system: let
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      default = pkgs.mkShell {
        inherit (self.checks.${system}.pre-commit-check) shellHook;
        env.GOFLAGS = "-buildvcs=false";
        packages = with pkgs; [
          go
          gopls
          golangci-lint
          gum
          just
          (writeShellScriptBin "test-minimal" (builtins.readFile ./scripts/dev/test-minimal.sh))
          (writeShellScriptBin "lint-minimal" (builtins.readFile ./scripts/dev/lint-minimal.sh))
          (writeShellScriptBin "ralph-stream.sh" (builtins.readFile ./scripts/ralph-stream.sh))
          (writeShellScriptBin "check-file-length" (builtins.readFile ./scripts/check-file-length.sh))
        ];
      };
    });
  };
}
