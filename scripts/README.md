# VPS Scripts

`install-vps.sh` is for first installation only.

It will:
- install required packages
- copy this project into `/opt/spark-whatsapp-module`
- create `/etc/spark-whatsapp-module.env`
- build the binary
- create and start a `systemd` service

Run it from the project root:

```bash
sudo bash scripts/install-vps.sh
```

After that, edit:

```bash
sudo nano /etc/spark-whatsapp-module.env
```

Then restart:

```bash
sudo systemctl restart spark-whatsapp-module
```

Normal updates after the first install are:

```bash
git pull
sudo systemctl restart spark-whatsapp-module
```
