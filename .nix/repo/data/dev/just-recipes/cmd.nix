{
  exec-cmd = {
    enable = true;
    groups = ["runners"];
    attributes = ["positional-arguments"];
    parameters = ["*ARGS"];
    dependencies = [''build-cmd''];
    recipe = ''git-workhours "$@"'';
  };

  build-cmd = {
    enable = true;
    groups = ["builders"];
    parameters = [''OUT="$PROJECT_ROOT/.cache/out/packages_git-workhours"''];
    dependencies = [''(build-nix "$PROJECT_ROOT/#git-workhours" "--out-link" OUT)''];
    recipe = ''
      cp -f "{{ OUT }}"/bin/* "$PROJECT_BIN"
    '';
  };
}
