{
  data,
  deps,
  lib,
  pkgs,
  synergy-lib,
  ...
} @ args: {
  just.recipes = synergy-lib.autoimport {
    inherit args;
    source = ./just-recipes;
    flatten = true;
    merge = true;
  };

  intellij-idea.file-watchers = lib.mkForce [
    {
      enabled = true;
      name = "Format";
      scope = "Current File";
      fileExtension = "*";
      workingDir = "$ProjectFileDir$";
      program = lib.getExe pkgs.just;
      arguments = "fmt $FileRelativePath$";
      output = "$FileRelativePath$";
    }
  ];

  nixago.intellij-idea-file-watchers = deps.synergy.result.lib.harmony.nixago.files.intellij-idea.file-watchers data.${pkgs.stdenv.hostPlatform.system}.dev.intellij-idea.file-watchers pkgs;

  git-cliff.enable = true;
}
