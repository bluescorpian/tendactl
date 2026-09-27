{
  description = "Command-line tool for Tenda routers";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      rev = self.shortRev or self.dirtyShortRev or "dev";
      forAll = f: nixpkgs.lib.genAttrs systems (s: f nixpkgs.legacyPackages.${s});
    in
    {
      packages = forAll (pkgs: {
        default = pkgs.buildGoModule {
          pname = "tendactl";
          version = rev;
          src = self;
          # Update when go.sum changes: set to pkgs.lib.fakeHash, build, copy "got:".
          vendorHash = "sha256-DVsmOLQ9MssT2yWX9M8osKrtL52nOm0lj6rC+P+etsk=";
          env.CGO_ENABLED = 0;
          ldflags = [ "-s" "-w" "-X github.com/bluescorpian/tendactl/cmd.version=${rev}" ];
          nativeBuildInputs = [ pkgs.installShellFiles ];
          postInstall = ''
            installShellCompletion --cmd tendactl \
              --bash <($out/bin/tendactl completion bash) \
              --zsh <($out/bin/tendactl completion zsh) \
              --fish <($out/bin/tendactl completion fish)
          '';
          meta = {
            description = "Command-line tool for Tenda routers";
            homepage = "https://github.com/bluescorpian/tendactl";
            license = pkgs.lib.licenses.mit;
            mainProgram = "tendactl";
          };
        };
      });

      devShells = forAll (pkgs: {
        default = pkgs.mkShell { packages = [ pkgs.go pkgs.goreleaser ]; };
      });
    };
}
