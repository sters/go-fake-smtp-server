package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sters/go-fake-smtp-server/fakesmtpserver"
	"golang.org/x/sync/errgroup"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx := context.Background()
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		slog.Info("smtp server", "error", fakesmtpserver.StartSMTPServer())

		return nil
	})
	eg.Go(func() error {
		slog.Info("view server", "error", fakesmtpserver.StartViewServer())

		return nil
	})

	// for local debug
	// eg.Go(func() error {
	// 	time.Sleep(time.Second * 3)
	//
	// 	for n := 0; n < 100; n++ {
	// 		c, err := smtp.Dial("localhost:10025")
	// 		if err != nil {
	// 			slog.Info("smtp client test", "error", err)
	// 			return err
	// 		}
	//
	// 		to := []string{"recipient@example.net"}
	// 		msg := strings.NewReader("To: recipient@example.net\r\n" +
	// 			"Subject: discount Gophers!\r\n" +
	// 			"\r\n" +
	// 			"This is the email body.\r\n")
	// 		err = c.SendMail("localhost:10025", to, msg)
	// 		if err != nil {
	// 			slog.Info("smtp client test", "error", err)
	// 		}
	//
	// 		time.Sleep(time.Second * 3)
	// 	}
	//
	// 	return nil
	// })

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt)
	select {
	case <-sigCh:
		slog.Info("Interrupt...")
	case <-ctx.Done():
	}
}
