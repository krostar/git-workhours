{deps, ...}:
deps.synergy.result.devShells.harmony.go.overrideAttrs (_: prev: {
  shellHook = ''
    ${prev.shellHook or ""}

    export PROJECT_BIN="$PROJECT_ROOT/.cache/bin";
    export PATH="$PROJECT_BIN:$PATH"
    mkdir -p "$PROJECT_BIN"
  '';
})
