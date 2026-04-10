{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  VERSION_PREFIX = "2.0.0";
  buildInputs = [
    pkgs.actionlint
    pkgs.azure-cli
    pkgs.dotnet-sdk
    pkgs.git
    pkgs.github-cli
    pkgs.gnumake
    pkgs.gnused
    pkgs.go_1_25
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
