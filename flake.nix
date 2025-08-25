{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    synergy = {
      url = "github:krostar/synergy";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = {synergy, ...} @ inputs:
    synergy.lib.mkFlake {
      inherit inputs;
      src = ./.nix;
      eval = {
        lib,
        synergy-lib,
        ...
      }: {
        synergy = {
          restrictDependenciesUnits.synergy = ["harmony"];
          export = {
            homeModules = hm: _: {inherit (hm.git-workhours) git-workhours;};
            packages = e:
              builtins.mapAttrs (
                _: units:
                  {inherit (units.git-workhours) git-workhours;}
                  // (synergy-lib.attrsets.liftChildren "-" (lib.attrsets.filterAttrs (k: _: k != "git-workhours") units))
              )
              e;
          };
        };
      };
    };
}
