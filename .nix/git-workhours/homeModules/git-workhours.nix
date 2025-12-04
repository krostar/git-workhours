{results, ...}: {
  config,
  lib,
  pkgs,
  ...
}: {
  meta.maintainers = with lib.maintainers; [krostar];

  options.programs.git-workhours = {
    enable = (lib.mkEnableOption "git-workhours, a set of git hooks to handle overtime") // {default = true;};
    package = lib.mkPackageOption results.systemized.${pkgs.stdenv.hostPlatform.system}.packages.git-workhours "git-workhours" {};
    schedule = lib.mkOption {
      type = lib.types.str;
      description = "work schedule, starting sunday";
      default = ",9h-19h,9h-19h,9h-19h,9h-19h,9h-19h,";
    };
  };

  config = let
    cfg = config.programs.git-workhours;
  in
    lib.mkIf cfg.enable {
      xdg.dataFile = {
        "git/hooks/pre-commit" = {
          executable = true;
          text = ''
            #!/usr/bin/env bash
            exec ${lib.getExe cfg.package} hooks pre-commit
          '';
        };
        "git/hooks/post-commit" = {
          executable = true;
          text = ''
            #!/usr/bin/env bash
            exec ${lib.getExe cfg.package} hooks post-commit
          '';
        };
        "git/hooks/pre-push" = {
          executable = true;
          text = ''
            #!/usr/bin/env bash
            exec ${lib.getExe cfg.package} hooks pre-push
          '';
        };
      };

      programs.git.settings = {
        core.hooksPath = config.xdg.dataHome + "/git/hooks/";
        wh.schedule = cfg.schedule;
      };
    };
}
