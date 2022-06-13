Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/focal64"

  config.vm.provision "shell" do |s|
    s.inline = <<-'SCRIPT'
echo '==> install golang'
cd /tmp
wget -q https://go.dev/dl/go1.18.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.18.3.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/golang.sh > /dev/null

echo '==> configure git'
git config --global --add safe.directory /vagrant
SCRIPT
  end
end
