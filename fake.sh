#!/bin/bash

# Fake installer script to test progress tracking
# This will output different installation steps with delays

# Get arguments
DISK=${1:-"/dev/sda"}
USERNAME=${2:-"kairos"}
PASSWORD=${3:-"password"}

# Print startup message
echo "Starting Kairos installation on disk $DISK"
echo "Using username: $USERNAME"
sleep 2

# Partitioning step
echo "Partitioning disk $DISK..."
sleep 3
echo "Creating partition table..."
sleep 1
echo "Creating boot partition..."
sleep 1
echo "Creating root partition..."
sleep 1
echo "Partitioning completed successfully."
sleep 2

# Formatting step
echo "Formatting partitions..."
sleep 2
echo "Formatting boot partition as FAT32..."
sleep 1
echo "Formatting root partition as EXT4..."
sleep 2
echo "Formatting completed."
sleep 2

# Installing base system
echo "Installing base system..."
sleep 3
echo "Extracting rootfs..."
sleep 2
echo "Setting up bootloader configuration..."
sleep 2
echo "Installing base packages..."
sleep 3
echo "Base system installation complete."
sleep 2

# Configuring
echo "Configuring bootloader..."
sleep 3
echo "Installing GRUB to $DISK..."
sleep 2
echo "Creating initial ramdisk..."
sleep 2
echo "Bootloader configuration complete."
sleep 2

# User setup
echo "Configuring user account..."
echo "Creating user $USERNAME..."
sleep 1
echo "Setting user password..."
sleep 1
echo "Configuring sudo access..."
sleep 1
echo "User account setup complete."
sleep 2

# SSH setup
if [ -n "$SSH_KEYS" ]; then
    echo "Setting up SSH keys..."
    sleep 2
    echo "SSH keys configured."
    sleep 1
fi

# Finalizing
echo "Finalizing installation..."
sleep 2
echo "Generating system configuration..."
sleep 1
echo "Running post-install scripts..."
sleep 2
echo "Syncing disks..."
sleep 2
echo "Finalizing complete."
sleep 2

# Complete
echo "Installation complete! System is ready to use."
echo "Success: Kairos has been installed on $DISK."

exit 0
