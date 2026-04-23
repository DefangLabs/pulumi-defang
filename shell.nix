{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  VERSION_PREFIX = "2.0.0";
  buildInputs = [
    pkgs.actionlint
    pkgs.azure-cli
    pkgs.dotnet-sdk_8
    pkgs.git
    pkgs.github-cli
    pkgs.gnumake
    pkgs.gnused
    pkgs.go_1_25
    pkgs.golangci-lint
    pkgs.less
    pkgs.nixfmt
    pkgs.nodejs_24
    pkgs.pulumi-bin
    pkgs.pulumictl
    (pkgs.python312.withPackages (ps: [
      ps.setuptools
    ]))
    pkgs.vim
    pkgs.yarn
  ];
}
