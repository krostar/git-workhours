{
  flake,
  pkgs,
  lib,
  ...
}: let
  isNotDirty = builtins.hasAttr "rev" flake.source;
in
  pkgs.buildGoModule {
    pname = "git-workhours";
    version = "0.0.1";

    subPackages = ["cmd"];
    src = let
      root = ../../..;
    in
      lib.fileset.toSource {
        inherit root;
        fileset = lib.fileset.unions [
          (root + "/cmd")
          (root + "/internal")
          (root + "/go.mod")
          (root + "/go.sum")
        ];
      };
    vendorHash = "sha256-8HNeuRF/IZocxpdOkxqjUFVSVQ+4lMt22fj3AvkqJrg=";

    ldflags = lib.lists.optionals isNotDirty ["-s" "-w"];
    doCheck = isNotDirty;

    postInstall = ''
      mv $out/bin/cmd $out/bin/git-workhours
    '';

    meta = {
      mainProgram = "git-workhours";
      maintainers = with lib.maintainers; [krostar];
    };
  }
