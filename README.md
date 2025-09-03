uinput-go

Small Go wrapper around Linux uinput to create a virtual keyboard device.

Notes:
- Requires Linux and /dev/uinput enabled in kernel.
- Requires root or capabilities to create uinput devices.

## Build

To build this app, linux environment is required because linux head import

```sh
sudo apt-get update && sudo apt-get install -y build-essential linux-headers-generic gcc-multilib libc6-dev-i386
```