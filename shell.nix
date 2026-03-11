{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  buildInputs = [
    pkgs.git
    pkgs.gnumake
    pkgs.gnused
    pkgs.go_1_24
    pkgs.less
    pkgs.nixfmt-classic
    pkgs.nodejs_22
    #pkgs.pulumi
    pkgs.pulumi-bin
    #pkgs.pulumiPackages.pulumi-language-dotnet
    #pkgs.pulumiPackages.pulumi-go
    #pkgs.pulumiPackages.pulumi-nodejs
    #pkgs.pulumiPackages.pulumi-python
    #pkgs.pulumiPackages.pulumi-language-yaml
    pkgs.pulumictl
    pkgs.python3
    pkgs.golangci-lint
    pkgs.python312Packages.setuptools
    pkgs.dotnet-sdk
    pkgs.vim
    pkgs.yarn
  ];
}
