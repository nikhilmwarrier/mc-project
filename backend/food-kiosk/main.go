package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"food-kiosk/backend"
	"food-kiosk/database"
	"food-kiosk/mqtt"
	"food-kiosk/server"

	qrcode "github.com/skip2/go-qrcode"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := server.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.Initialize(db); err != nil {
		log.Fatal(err)
	}

	if cfg.AdminPrincipal != "" {
		if err := server.BootstrapAdmin(context.Background(), db, cfg.AdminPrincipal, cfg.AdminPassword); err != nil {
			log.Fatalf("bootstrap admin: %v", err)
		}
		log.Printf("admin credential %q is ready", cfg.AdminPrincipal)
	}

	queues, err := backend.New(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	client, err := mqtt.New(ctx, mqtt.Config{
		Broker:   server.Getenv("MQTT_BROKER", "tcp://127.0.0.1:1883"),
		ClientID: server.Getenv("MQTT_CLIENT_ID", "canteen-backend"),
		Username: os.Getenv("MQTT_USERNAME"), Password: os.Getenv("MQTT_PASSWORD"),
	}, queues, log.Default())
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Println("canteen MQTT service running")
		if err := client.Run(); err != nil {
			log.Fatalf("mqtt: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.NewRouter(cfg, db, queues),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	url := publicURL(cfg.Addr)
	log.Printf("frontend + API: %s", url)
	if cfg.WebDir != "" {
		if _, err := os.Stat(cfg.WebDir); err != nil {
			log.Printf("frontend not built at %s (run `just serve`)", cfg.WebDir)
		}
	}
	if qr, err := qrcode.New(url, qrcode.Medium); err == nil {
		fmt.Println(qr.ToSmallString(false))
	}

	// Run the server in the background so the main goroutine can wait for a
	// signal and shut them both down cleanly.
	go func() {
		log.Printf("food-kiosk server listening on %s (auth required: %t)", cfg.Addr, cfg.AuthRequired)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}

// publicURL builds the address printed in the QR code. A wildcard bind is
// replaced with the laptop's LAN IPv4 so phones on the same Wi-Fi can reach it.
func publicURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = "8080"
	}
	if override := os.Getenv("PUBLIC_HOST"); override != "" {
		host = override
	} else if host == "" || host == "0.0.0.0" || host == "::" {
		if ip := lanIPv4(); ip != nil {
			host = ip.String()
		} else {
			host = "127.0.0.1"
		}
	}
	return "http://" + net.JoinHostPort(host, port)
}

// lanIPv4 prefers en0 then en1 (the same interfaces the flash recipe uses),
// then falls back to the default route and finally any non-loopback IPv4.
func lanIPv4() net.IP {
	for _, name := range []string{"en0", "en1"} {
		if iface, err := net.InterfaceByName(name); err == nil {
			if ip := firstIPv4(iface); ip != nil {
				return ip
			}
		}
	}

	if conn, err := net.Dial("udp", "8.8.8.8:80"); err == nil {
		defer conn.Close()
		if udp, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			if ip := udp.IP.To4(); ip != nil && !ip.IsLoopback() {
				return ip
			}
		}
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for i := range ifaces {
		if ip := firstIPv4(&ifaces[i]); ip != nil {
			return ip
		}
	}
	return nil
}

func firstIPv4(iface *net.Interface) net.IP {
	if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return nil
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip4 := ip.To4(); ip4 != nil && !ip4.IsLoopback() && !ip4.IsLinkLocalUnicast() {
			return ip4
		}
	}
	return nil
}
