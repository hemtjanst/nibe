package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"

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

	errOut := flag.CommandLine.Output()

	if *user == "" || *passwd == "" || *fingerprint == "" || *serial == "" || *endpoint == "" {
		fmt.Fprintln(errOut, "all flags must be provided")
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
			fmt.Fprintln(errOut, err)
			os.Exit(1)
		}

		if err := output(os.Stdout, devs, *format); err != nil {
			fmt.Fprintln(errOut, err)
			os.Exit(1)
		}
	case "device":
		if len(args) != 2 {
			fmt.Fprintln(errOut, "device needs an ID")
			os.Exit(1)
		}

		dev, err := client.Device(ctx, args[1])
		if err != nil {
			fmt.Fprintln(errOut, err)
			os.Exit(1)
		}

		if err := output(os.Stdout, dev, *format); err != nil {
			fmt.Fprintln(errOut, err)
			os.Exit(1)
		}
	case "points":
		if len(args) != 2 {
			fmt.Fprintln(errOut, "points needs a device ID")
			os.Exit(1)
		}

		points, err := client.Points(ctx, args[1])
		if err != nil {
			fmt.Fprintln(errOut, err)
			os.Exit(1)
		}

		if err := output(os.Stdout, points, *format); err != nil {
			fmt.Fprintln(errOut, err)
			os.Exit(1)
		}
	case "point":
		switch len(args) {
		case 3:
			point, err := client.Point(ctx, args[1], args[2])
			if err != nil {
				fmt.Fprintln(errOut, err)
				os.Exit(1)
			}

			if err := output(os.Stdout, point, *format); err != nil {
				fmt.Fprintln(errOut, err)
				os.Exit(1)
			}
		case 4:
			point, err := client.Point(ctx, args[1], args[2])
			if err != nil {
				fmt.Fprintln(errOut, err)
				os.Exit(1)
			}

			if !point.Metadata.Writable {
				fmt.Fprintf(errOut, "variable: %d (%s) is not writable\n", point.Metadata.VariableID, point.Title)
				os.Exit(1)
			}

			nv := point.Value

			switch point.Metadata.VariableType {
			case nibe.VariableTypeInteger:
				value, err := strconv.ParseInt(args[3], 10, 32)
				if err != nil {
					fmt.Fprintf(errOut, "variable: %d (%s) must be %s\n", point.Metadata.VariableID, point.Title, point.Metadata.VariableType)
					os.Exit(1)
				}

				nv.Int = int(value)
			default:
				fmt.Fprintln(errOut, "cannot change points other than ints")
				os.Exit(1)
			}

			patched, err := client.PatchPoints(ctx, args[1], nv)
			if err != nil {
				fmt.Fprintln(errOut, err)
				os.Exit(1)
			}

			if err := output(os.Stdout, patched, *format); err != nil {
				fmt.Fprintln(errOut, err)
				os.Exit(1)
			}
		default:
			fmt.Fprintln(errOut, "point needs a device ID and a variable ID, and an optional value to set it to")
			os.Exit(1)
		}
	case "notifications":
		if len(args) != 2 {
			fmt.Fprintln(errOut, "notifications needs a device ID")
			os.Exit(1)
		}

		notifs, err := client.Notifications(ctx, args[1])
		if err != nil {
			fmt.Fprintln(errOut, err)
			os.Exit(1)
		}

		if err := output(os.Stdout, notifs, *format); err != nil {
			fmt.Fprintln(errOut, err)
			os.Exit(1)
		}
	case "notifications-reset":
		if len(args) != 2 {
			fmt.Fprintln(errOut, "notifications-reset needs a device ID")
			os.Exit(1)
		}

		if err := client.ResetNotifications(ctx, args[1]); err != nil {
			fmt.Fprintln(errOut, err)
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
