package eventstream

// type LocalConsumer struct {
// }

// type HistoricalConsumer struct {
// }

// type Consumer struct {
// 	Journals journal.Store[*journalpb.Record]

// 	once                   sync.Once
// 	subscribe, unsubscribe chan *subscriber

// 	done          signaling.Latch
// 	workers       uuidpb.Map[*consumerWorker]
// 	workerStopped chan workerResult
// }

// type subscriber struct {
// 	StreamID *uuidpb.UUID
// 	Offset   Offset
// 	Events   chan<- Event
// 	Desync   signaling.Event
// }

// func (c *Consumer) Run(ctx context.Context) error {
// 	c.init()
// 	defer c.done.Signal()
// 	return fsm.Start(ctx, c.idleState)
// }

// func (c *Consumer) init() {
// 	c.once.Do(func() {
// 		c.subscribe = make(chan *subscriber)
// 		c.unsubscribe = make(chan *subscriber)
// 	})
// }

// func (c *Consumer) Consume(
// 	ctx context.Context,
// 	streamID *uuidpb.UUID,
// 	begin Offset,
// 	events chan<- Event,
// ) error {
// 	c.init()

// 	sub := &subscriber{
// 		StreamID: streamID,
// 		Events:   events,
// 		Offset:   begin,
// 	}

// 	for {
// 		if err := c.consumeHistorical(ctx, sub); err != nil {
// 			return err
// 		}
// 		if err := c.consumeContemporary(ctx, sub); err != nil {
// 			return err
// 		}
// 	}
// }

// func (c *Consumer) consumeHistorical(ctx context.Context, sub *subscriber) error {
// 	j, err := c.Journals.Open(ctx, journalName(sub.StreamID))
// 	if err != nil {
// 		return err
// 	}
// 	defer j.Close()

// 	begin, end, err := j.Bounds(ctx)
// 	if err != nil {
// 		return err
// 	}

// 	return journal.RangeFromSearchResult(
// 		ctx,
// 		j,
// 		begin,
// 		end,
// 		searchByOffset(sub.Offset),
// 		func(
// 			ctx context.Context,
// 			_ journal.Position,
// 			rec *journalpb.Record,
// 		) (bool, error) {
// 			offset := Offset(rec.StreamOffsetBefore)

// 			for _, env := range rec.GetEventsAppended().GetEvents() {
// 				select {
// 				case <-ctx.Done():
// 					return false, ctx.Err()
// 				case sub.Events <- Event{
// 					StreamID: sub.StreamID,
// 					Offset:   offset,
// 					Envelope: env,
// 				}:
// 					offset++
// 				}
// 			}

// 			return true, nil
// 		},
// 	)
// }

// func (c *Consumer) consumeContemporary(ctx context.Context, sub *subscriber) error {
// 	select {
// 	case <-ctx.Done():
// 		return ctx.Err()
// 	case <-c.done:
// 		return errors.New("consumer has stopped")
// 	case c.subscribe <- sub:
// 		// subscription request received
// 	}

// 	defer func() {
// 		select {
// 		case <-c.done:
// 		case c.unsubscribe <- sub:
// 		}
// 	}()

// 	select {
// 	case <-ctx.Done():
// 		return ctx.Err()
// 	case <-c.done:
// 		return errors.New("consumer has stopped")
// 	case <-sub.Desync.Signaled():
// 		return nil
// 	}
// }

// func (c *Consumer) idleState(ctx context.Context) fsm.Action {
// 	select {
// 	case <-ctx.Done():
// 		return fsm.Stop()
// 	// case e := <-c.Events:
// 	// 	return fsm.With(e).EnterState(c.dispatchState)
// 	case sub := <-c.subscribe:
// 		return fsm.With(sub).EnterState(c.subscribeState)
// 	case sub := <-c.unsubscribe:
// 		return fsm.With(sub).EnterState(c.unsubscribeState)
// 	}
// }

// func (c *Consumer) subscribeState(_ context.Context, sub *subscriber) fsm.Action {
// 	// if c.offset == nil || con.Offset == *c.offset {
// 	// 	c.subscribers[con] = struct{}{}
// 	// } else {
// 	// 	con.Desynced <- struct{}{}
// 	// }
// 	return fsm.EnterState(c.idleState)
// }

// func (c *Consumer) unsubscribeState(_ context.Context, sub *subscriber) fsm.Action {
// 	// if c.offset == nil || con.Offset == *c.offset {
// 	// 	c.subscribers[con] = struct{}{}
// 	// } else {
// 	// 	con.Desynced <- struct{}{}
// 	// }
// 	return fsm.EnterState(c.idleState)
// }

// // func (c *Consumer) dispatchState(_ context.Context, e Event) fsm.Action {
// // 	for con := range c.subscribers {
// // 		select {
// // 		case con.Events <- e:
// // 		default:
// // 			close(con.Desynced)
// // 			delete(c.subscribers, con)
// // 		}
// // 	}
// // 	return fsm.EnterState(c.idleState)
// // }

// // workerByStreamID returns the worker that manages the state of the stream with
// // the given ID. The worker is started if it is not already running.
// func (c *Consumer) workerByStreamID(
// 	ctx context.Context,
// 	streamID *uuidpb.UUID,
// ) (*consumerWorker, error) {
// 	if w, ok := c.workers.TryGet(streamID); ok {
// 		return w, nil
// 	}

// 	w, err := c.startWorkerForStreamID(ctx, streamID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	c.workers.Set(streamID, w)

// 	return w, nil
// }

// // startWorkerForStreamID starts a new worker for the stream with the given ID.
// func (c *Consumer) startWorkerForStreamID(
// 	ctx context.Context,
// 	streamID *uuidpb.UUID,
// ) (*consumerWorker, error) {
// 	// j, err := c.journals.Open(ctx, journalName(streamID))
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	w := &consumerWorker{
// 		// Journal: j,
// 		// Events:  c.Events,
// 		// Logger: c.Logger.With(
// 		// 	slog.String("stream_id", streamID.AsString()),
// 		// ),
// 	}

// 	go func() {
// 		// 	defer j.Close()

// 		c.workerStopped <- workerResult{
// 			StreamID: streamID,
// 			Err:      w.Run(ctx),
// 		}
// 	}()

// 	return w, nil
// }

// type consumerWorker struct{}
