package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/sters/go-fake-smtp-server/fakesmtpserver"
)

func main() {
	lgr, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(lgr)

	ctx := context.Background()
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		zap.L().Info("smtp server", zap.Error(fakesmtpserver.StartSmtpServer()))
		return nil
	})
	eg.Go(func() error {
		zap.L().Info("view server", zap.Error(fakesmtpserver.StartViewServer()))
		return nil
	})

	// for local debug
	// eg.Go(func() error {
	// 	time.Sleep(time.Second * 3)
	//
	// 	for n := 0; n < 100; n++ {
	// 		c, err := smtp.Dial("localhost:10025")
	// 		if err != nil {
	// 			zap.L().Info("smtp client test", zap.Error(err))
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
	// 			zap.L().Info("smtp client test", zap.Error(err))
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
		zap.L().Info("Interrupt...")
	case <-ctx.Done():
	}
}
