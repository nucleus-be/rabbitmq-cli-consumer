package consumer

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"github.com/ricbra/rabbitmq-cli-consumer/command"
	"github.com/ricbra/rabbitmq-cli-consumer/config"
	"github.com/streadway/amqp"
	"log"
	"net/url"
	"strconv"
	"time"
)

type Consumer struct {
	Channel     *amqp.Channel
	Connection  *amqp.Connection
	Queue       string
	Factory     command.Factory
	ErrLogger   *log.Logger
	InfLogger   *log.Logger
	DeadLetter  bool
	Retry       int
	Runner      command.Runner
	Compression bool
}

func (c *Consumer) Consume() {
	c.InfLogger.Println("Registering consumer... ")
	msgs, err := c.Channel.Consume(c.Queue, "", false, false, false, false, nil)
	if err != nil {
		c.ErrLogger.Fatalf("Failed to register a consumer: %s", err)
	}
	c.InfLogger.Println("Succeeded registering consumer.")

	var sendCh *amqp.Channel

	if c.DeadLetter {
		var err error
		sendCh, err = c.Connection.Channel()
		if err != nil {
			c.ErrLogger.Println("Could not open channel to republish failed jobs %s", err)
		}
		defer sendCh.Close()
	}

	defer c.Connection.Close()
	defer c.Channel.Close()

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			c.InfLogger.Println("reading deliveries")
			go func(delivery amqp.Delivery) {
				input := delivery.Body
				if c.Compression {
					c.InfLogger.Println("Compressed message")
					decompressed, err := c.Decompress(input)
					if err != nil {
						c.ErrLogger.Println("Could not create zlib handler")
						delivery.Nack(true, true)
					}
					input = decompressed

				}
				if c.DeadLetter {
					c.DeadLetterDelivery(delivery, input, sendCh)
				} else {
					c.NormalDelivery(delivery, input)
				}

			}(d)

		}
	}()
	c.InfLogger.Println("Waiting for messages...")
	<-forever
}

func (c *Consumer) DeadLetterDelivery(d amqp.Delivery, input []byte, sendCh *amqp.Channel) {
	if d.Headers == nil {
		d.Headers = make(map[string]interface{}, 0)
	}
	retry, ok := d.Headers["retry_count"]
	if !ok {
		retry = "0"
	}
	c.InfLogger.Println(fmt.Sprintf("retry %s", retry))

	retryCount, err := strconv.Atoi(retry.(string))
	if err != nil {
		c.ErrLogger.Fatal("could not parse retry header")
	}

	c.InfLogger.Println(fmt.Sprintf("retryCount : %d max retries: %d", retryCount, c.Retry))

	cmd := c.Factory.Create(input)
	if c.Runner.Run(cmd) {
		d.Ack(true)
	} else if retryCount >= c.Retry {
		d.Nack(true, false)
	} else {
		//republish message with new retry header
		retryCount++
		d.Headers["retry_count"] = strconv.Itoa(retryCount)
		republish := amqp.Publishing{
			ContentType:     d.ContentType,
			ContentEncoding: d.ContentEncoding,
			Timestamp:       time.Now(),
			Body:            d.Body,
			Headers:         d.Headers,
		}
		err = sendCh.Publish("", c.Queue, false, false, republish)
		if err != nil {
			c.ErrLogger.Println("error republish %s", err)
		}
		d.Ack(true)
	}
}

func (c *Consumer) NormalDelivery(d amqp.Delivery, input []byte) {
	cmd := c.Factory.Create(input)
	if c.Runner.Run(cmd) {
		d.Ack(true)
	} else {
		d.Nack(true, false)
	}
}

func (c *Consumer) Decompress(input []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := zlib.NewWriterLevel(&b, zlib.BestCompression)
	if err != nil {
		return []byte{}, err
	}
	w.Write(input)
	w.Close()

	return b.Bytes(), nil

}

func New(cfg *config.Config, factory command.Factory, errLogger, infLogger *log.Logger) (*Consumer, error) {
	uri := fmt.Sprintf(
		"amqp://%s:%s@%s:%s%s",
		url.QueryEscape(cfg.RabbitMq.Username),
		url.QueryEscape(cfg.RabbitMq.Password),
		cfg.RabbitMq.Host,
		cfg.RabbitMq.Port,
		cfg.RabbitMq.Vhost,
	)

	infLogger.Println("Connecting RabbitMQ...")
	conn, err := amqp.Dial(uri)
	if nil != err {
		return nil, errors.New(fmt.Sprintf("Failed connecting RabbitMQ: %s", err.Error()))
	}
	infLogger.Println("Connected.")

	infLogger.Println("Opening channel...")
	ch, err := conn.Channel()
	if nil != err {
		return nil, errors.New(fmt.Sprintf("Failed to open a channel: %s", err.Error()))
	}
	infLogger.Println("Done.")

	infLogger.Println("Setting QoS... ")
	// Attempt to preserve BC here
	if cfg.Prefetch.Count == 0 {
		cfg.Prefetch.Count = 3
	}
	if err := ch.Qos(cfg.Prefetch.Count, 0, cfg.Prefetch.Global); err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to set QoS: %s", err.Error()))
	}
	infLogger.Println("Succeeded setting QoS.")

	// Check for missing exchange settings to preserve BC
	if "" == cfg.Exchange.Name && "" == cfg.Exchange.Type && !cfg.Exchange.Durable && !cfg.Exchange.Autodelete {
		cfg.Exchange.Type = "direct"
	}

	var table map[string]interface{}
	deadLetter := false

	if "" != cfg.Deadexchange.Name {
		infLogger.Printf("Declaring exchange \"%s\"...", cfg.Deadexchange.Name)
		err = ch.ExchangeDeclare(cfg.Deadexchange.Name, cfg.Deadexchange.Type, cfg.Deadexchange.Durable, cfg.Deadexchange.AutoDelete, false, false, amqp.Table{})

		if nil != err {
			return nil, errors.New(fmt.Sprintf("Failed to declare exchange: %s", err.Error()))
		}

		table = make(map[string]interface{}, 0)
		table["x-dead-letter-exchange"] = cfg.Deadexchange.Name

		infLogger.Printf("Declaring error queue \"%s\"...", cfg.Deadexchange.Queue)
		_, err = ch.QueueDeclare(cfg.Deadexchange.Queue, true, false, false, false, amqp.Table{})

		// Bind queue
		infLogger.Printf("Binding  error queue \"%s\" to dead letter exchange \"%s\"...", cfg.Deadexchange.Queue, cfg.Deadexchange.Name)
		err = ch.QueueBind(cfg.Deadexchange.Queue, "", cfg.Deadexchange.Name, false, amqp.Table{})

		if nil != err {
			return nil, errors.New(fmt.Sprintf("Failed to bind queue to exchange: %s", err.Error()))
		}
		deadLetter = true
	}

	// Empty Exchange name means default, no need to declare
	if "" != cfg.Exchange.Name {
		infLogger.Printf("Declaring deadletter exchange \"%s\"...", cfg.Exchange.Name)
		err = ch.ExchangeDeclare(cfg.Exchange.Name, cfg.Exchange.Type, cfg.Exchange.Durable, cfg.Exchange.Autodelete, false, false, amqp.Table{})

		if nil != err {
			return nil, errors.New(fmt.Sprintf("Failed to declare deadletter exchange: %s", err.Error()))
		}

		// Bind queue (before declare??)
		infLogger.Printf("Binding queue \"%s\" to deadletter exchange \"%s\"...", cfg.RabbitMq.Queue, cfg.Exchange.Name)
		err = ch.QueueBind(cfg.RabbitMq.Queue, "", cfg.Exchange.Name, false, table)

		if nil != err {
			return nil, errors.New(fmt.Sprintf("Failed to bind queue to deadletter exchange: %s", err.Error()))
		}
	}

	infLogger.Printf("Declaring queue \"%s\"...with args: %+v", cfg.RabbitMq.Queue, table)
	_, err = ch.QueueDeclare(cfg.RabbitMq.Queue, true, false, false, false, table)

	if nil != err {
		return nil, errors.New(fmt.Sprintf("Failed to declare queue: %s", err.Error()))
	}

	return &Consumer{
		Channel:     ch,
		Connection:  conn,
		Queue:       cfg.RabbitMq.Queue,
		Factory:     factory,
		ErrLogger:   errLogger,
		InfLogger:   infLogger,
		Runner:      command.NewRunner(errLogger, infLogger),
		Compression: cfg.RabbitMq.Compression,
		DeadLetter:  deadLetter,
		Retry:       cfg.Deadexchange.Retry,
	}, nil
}
