{
  description = "Meridian — a cross-platform network engineer's toolkit";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        lib = pkgs.lib;

        # Bump per release (or wire this to a VERSION file). Only stamps the
        # --version string; it does not affect which commit is built.
        version = "1.18.2";

        # The module targets Go 1.26, so pin the toolchain rather than taking
        # whatever nixpkgs defaults to.
        buildGoModule = pkgs.buildGoModule.override { go = pkgs.go_1_26; };

        # Shared across the CLI and the GUI — same go.mod / go.sum.
        #
        # FINALIZE: this is a placeholder. On a machine with Nix, run
        #   nix build .#cli
        # it will fail with "hash mismatch ... got: sha256-…"; paste that value
        # here. (Both packages reuse it.)
        vendorHash = lib.fakeHash;

        ldflags = [
          "-s"
          "-w"
          "-X"
          "github.com/hriday/meridian/internal/version.Version=v${version}"
        ];

        # ---- headless CLI: pure Go, CGO off, no frontend --------------------
        # Builds on NixOS, nix-darwin, and any Linux. This is the default
        # package, so `nix run github:hriday/meridian` runs the CLI.
        meridian-cli = buildGoModule {
          pname = "meridian-cli";
          inherit version vendorHash ldflags;
          src = self;
          subPackages = [ "cmd/meridian-cli" ];
          env.CGO_ENABLED = 0;
          doCheck = false;
          meta = {
            description = "Meridian network toolkit (command line)";
            homepage = "https://github.com/hriday/meridian";
            mainProgram = "meridian-cli";
            platforms = lib.platforms.unix;
          };
        };

        # ---- frontend (Svelte + Vite) → dist, for the GUI only -------------
        # FINALIZE: run `nix build .#gui`; it will fail with the correct
        # npmDepsHash — paste it here. (Not needed for the CLI.)
        meridian-frontend = pkgs.buildNpmPackage {
          pname = "meridian-frontend";
          inherit version;
          src = self + "/frontend";
          npmDepsHash = lib.fakeHash;
          installPhase = ''
            runHook preInstall
            cp -r dist $out
            runHook postInstall
          '';
        };

        # ---- Wails desktop GUI: Linux only (webkit2gtk), CGO on ------------
        # main.go does //go:embed all:frontend/dist, so the built frontend is
        # copied into place before the Go build.
        meridian-gui = buildGoModule {
          pname = "meridian";
          inherit version vendorHash ldflags;
          src = self;
          subPackages = [ "." ];
          tags = [ "webkit2_41" ];
          nativeBuildInputs = [
            pkgs.pkg-config
            pkgs.wrapGAppsHook3
          ];
          buildInputs = [
            pkgs.gtk3
            pkgs.webkitgtk_4_1
          ];
          preBuild = ''
            rm -rf frontend/dist
            cp -r ${meridian-frontend} frontend/dist
          '';
          doCheck = false;
          meta = {
            description = "Meridian network toolkit (desktop GUI)";
            homepage = "https://github.com/hriday/meridian";
            mainProgram = "meridian";
            platforms = lib.platforms.linux;
          };
        };
      in
      {
        packages =
          {
            default = meridian-cli;
            cli = meridian-cli;
          }
          // lib.optionalAttrs pkgs.stdenv.isLinux {
            gui = meridian-gui;
          };

        apps.default = {
          type = "app";
          program = "${meridian-cli}/bin/meridian-cli";
        };
      }
    );
}
