package ocd

import (
	"anodik/internal/process"
	"fmt"
)

func StopInterface(manager *process.Manager, iface string) error {
	session := SessionPath(iface)
	if !manager.IsAlive(session) {
		return nil
	}

	fmt.Printf("Another OpenOCD is already using interface %s. Stopping it...\n", iface)
	if err := manager.Kill(session); err != nil {
		return fmt.Errorf("cannot stop OpenOCD on interface %s: %w", iface, err)
	}
	fmt.Println("Stopped successfully.")
	return nil
}
