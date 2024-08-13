{
  #inputs = {
  #  nixpkgs.url = "github:NixOS/nixpkgs/eabe8d3eface69f5bb16c18f8662a702f50c20d5";#41680f83ea143f3ce4bd774ff53cca91f1dac826";#8ba91230a0da929172cf003e7b04efd6dd61b314";
  #};
  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = import nixpkgs {
            inherit system;
          };
        in
        {
          devShells.default = import ./shell.nix { pkgs = pkgs; };
        }
      );
}
