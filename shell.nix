{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  VERSION_PREFIX = "2.0.0";
  buildInputs = [
    pkgs.azure-cli
    pkgs.dotnet-sdk
    pkgs.git
    pkgs.gnumake
    pkgs.gnused
    pkgs.go_1_26
    pkgs.golangci-lint
    pkgs.less
    pkgs.nixfmt-classic
    pkgs.nodejs_22
    pkgs.pulumi-bin
    pkgs.pulumictl
    (pkgs.python312.withPackages (ps: [
      ps.setuptools
    ]))
    pkgs.vim
    pkgs.yarn
  ];
}
