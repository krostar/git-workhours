{
  testers.go.enable = true;

  linters = {
    commitlint.enable = true;
    golangci-lint = {
      enable = true;
      linters = {
        exclusions = {
          rules = [
            {
              path = "cmd/handler/hooks";
              linters = ["revive"];
              text = "package-directory-mismatch";
            }
            {
              path = "internal/git/config";
              linters = ["revive"];
              text = "package-directory-mismatch";
            }
            {
              path = "internal/git/config/reflectx";
              linters = ["depguard"];
              text = "import 'reflect' is not allowed";
            }
            {
              path = "internal/git/config/reflectx";
              linters = ["gosec"];
              text = "Use of unsafe calls should be audited";
            }
            {
              path = "cmd/handler/hooks/post-commit";
              linters = ["gosec"];
              text = "G404: Use of weak random number generator";
            }
            {
              path = "internal/git/";
              linters = ["gosec"];
              text = "G204: Subprocess launched with a potential tainted input or cmd arguments|G204: Subprocess launched with variable";
            }
            {
              path = "internal/workhours/workhours\.go";
              linters = ["gocritic"];
              text = "hugeParam: ws is heavy";
            }
          ];
        };
      };
    };
  };
}
