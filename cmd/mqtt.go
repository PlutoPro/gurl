package cmd

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/cobra"
)

var (
	mqttBroker, mqttUser, mqttPass, mqttClientID string
	mqttQoS                                      int
	mqttRetain, mqttInsecure                     bool
	mqttTimeout                                  time.Duration
	mqttCount                                    int
)

var mqttCmd = &cobra.Command{
	Use:   "mqtt",
	Short: "Publish or subscribe to MQTT topics",
}

func mqttConnect() (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(mqttBroker).
		SetClientID(mqttClientID).
		SetUsername(mqttUser).
		SetPassword(mqttPass).
		SetConnectTimeout(mqttTimeout).
		SetAutoReconnect(false).
		SetTLSConfig(&tls.Config{InsecureSkipVerify: mqttInsecure})
	c := mqtt.NewClient(opts)
	t := c.Connect()
	if !t.WaitTimeout(mqttTimeout) {
		return nil, fmt.Errorf("connect to %s timed out", mqttBroker)
	}
	if err := t.Error(); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return c, nil
}

var mqttPubCmd = &cobra.Command{
	Use:   "pub <topic> <message>",
	Short: "Publish a message",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := mqttConnect()
		if err != nil {
			return err
		}
		defer c.Disconnect(250)
		t := c.Publish(args[0], byte(mqttQoS), mqttRetain, args[1])
		if !t.WaitTimeout(mqttTimeout) {
			return fmt.Errorf("publish timed out")
		}
		if err := t.Error(); err != nil {
			return err
		}
		fmt.Printf("Published to %s\n", args[0])
		return nil
	},
}

var mqttSubCmd = &cobra.Command{
	Use:   "sub <topic>",
	Short: "Subscribe and print messages (Ctrl+C or --count N to stop)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := mqttConnect()
		if err != nil {
			return err
		}
		defer c.Disconnect(250)

		got := make(chan struct{}, 64)
		t := c.Subscribe(args[0], byte(mqttQoS), func(_ mqtt.Client, m mqtt.Message) {
			fmt.Printf("%s: %s\n", m.Topic(), m.Payload())
			got <- struct{}{}
		})
		if !t.WaitTimeout(mqttTimeout) {
			return fmt.Errorf("subscribe timed out")
		}
		if err := t.Error(); err != nil {
			return err
		}

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		for n := 0; ; {
			select {
			case <-got:
				n++
				if mqttCount > 0 && n >= mqttCount {
					return nil
				}
			case <-sig:
				return nil
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(mqttCmd)
	pf := mqttCmd.PersistentFlags()
	pf.StringVarP(&mqttBroker, "broker", "b", "tcp://localhost:1883", "Broker URL (tcp://, ssl://, ws://)")
	pf.StringVar(&mqttUser, "user", "", "Username")
	pf.StringVar(&mqttPass, "pass", "", "Password")
	pf.StringVar(&mqttClientID, "client-id", fmt.Sprintf("gurl-%d", os.Getpid()), "Client ID")
	pf.IntVarP(&mqttQoS, "qos", "q", 0, "QoS level (0, 1, 2)")
	pf.BoolVar(&mqttInsecure, "insecure", false, "Skip TLS certificate verification")
	pf.DurationVar(&mqttTimeout, "timeout", 10*time.Second, "Connect/operation timeout")
	mqttPubCmd.Flags().BoolVar(&mqttRetain, "retain", false, "Set the retain flag")
	mqttSubCmd.Flags().IntVarP(&mqttCount, "count", "n", 0, "Exit after N messages (0 = run until Ctrl+C)")
	mqttCmd.AddCommand(mqttPubCmd, mqttSubCmd)
}
