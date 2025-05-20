{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  buildInputs = [
    pkgs.git
    pkgs.gnumake
    pkgs.gnused
    pkgs.go_1_23
    pkgs.less
    pkgs.nixfmt-classic
    pkgs.nodejs_22
    pkgs.pulumi-bin
    pkgs.pulumictl
    pkgs.python3
    pkgs.golangci-lint
    pkgs.python312Packages.setuptools
    pkgs.dotnet-sdk
    pkgs.vim
    pkgs.yarn
  ];
}
