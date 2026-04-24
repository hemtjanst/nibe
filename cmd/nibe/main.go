package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"hemtjan.st/nibe"
)

func main() {
	user := flag.String("username", "", "username from menu 7.5.15")
	passwd := flag.String("password", "", "password from menu 7.5.15")
	fingerprint := flag.String("fingerprint", "", "fingerprint from menu 7.5.15")
	endpoint := flag.String("endpoint", "", "https://IP:8443 or https://hostname:8443")
	serial := flag.String("serial", "", "device serial number")
	format := flag.String("format", "json", "output format: json, go")

	flag.Parse()

	if *passwd == "" {
		*passwd = os.Getenv("NIBE_PASSWORD")
	}

	if *user == "" || *passwd == "" || *fingerprint == "" || *serial == "" || *endpoint == "" {
		fmt.Println("all flags must be provided")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background())
	defer cancel()

	client := nibe.New(
		nibe.WithEndpoint(*endpoint),
		nibe.WithUser(*user),
		nibe.WithPassword(*passwd),
		nibe.WithFingerprint(*fingerprint),
		nibe.WithSerial(*serial),
	)

	args := flag.Args()
	if len(args) == 0 {
		os.Exit(1)
	}

	switch args[0] {
	case "devices":
		devs, err := client.Devices(ctx)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		if err := output(os.Stdout, devs, *format); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	case "device":
		if len(args) != 2 {
			fmt.Fprintf(os.Stderr, "device needs an ID")
			os.Exit(1)
		}

		dev, err := client.Device(ctx, args[1])
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		if err := output(os.Stdout, dev, *format); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	case "points":
		if len(args) != 2 {
			fmt.Fprintf(os.Stderr, "points needs a device ID")
			os.Exit(1)
		}

		points, err := client.Points(ctx, args[1])
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		if err := output(os.Stdout, points, *format); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	case "point":
		if len(args) != 3 {
			fmt.Fprintf(os.Stderr, "point needs a device ID and a variable ID")
			os.Exit(1)
		}

		point, err := client.Point(ctx, args[1], args[2])
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		if err := output(os.Stdout, point, *format); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func output(w io.Writer, data any, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		return enc.Encode(data)
	case "go":
		switch v := data.(type) {
		case map[string]nibe.Point:
			for _, elem := range v {
				_, err := fmt.Fprintf(w, "%#v\n", elem)
				if err != nil {
					return err
				}
			}
		case []nibe.Device:
			for _, elem := range v {
				_, err := fmt.Fprintf(w, "%#v\n", elem)
				if err != nil {
					return err
				}
			}
		default:
			_, err := fmt.Fprintf(w, "%#v\n", data)
			return err
		}
	default:
		return fmt.Errorf("invalid format value")
	}

	return nil
}
