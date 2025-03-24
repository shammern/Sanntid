Implementation roadmap

1. Define the Overall Architecture and Message Protocol
    a. Requirements and Design Decisions
        Elevator Roles:
            Master: Delegates orders and aggregates state.
            Slaves: Execute orders and continuously report their status.
        Global State:
            All nodes maintain a synchronized view of the elevator statuses.
        Transport & Reliability:
            Use UDP for most communications initially.
            Implement measures to overcome UDP’s unreliability (packet loss, reordering).
            Plan for a future TCP channel for heartbeats and critical messages.
    b. Message Protocol
        Message Types: Define separate message types (e.g., "stateUpdate", "order", "heartbeat", "ack").
        JSON Format:
            Create a standard JSON schema for all messages. A typical message might include:
            Type (string): The kind of message.
            Seq (uint64): A sequence number to detect missing or out-of-order messages.
            Timestamp (time.Time): When the message was sent.
            Payload (object): Message-specific data.

3. Implement the Transport Layer (UDP)
    a. UDP Listener (Server)
        Open a UDP Socket: Create a function to bind to a specific port.
        Read Incoming Packets: Read data from the UDP socket. Decide on your message framing (e.g., newline-delimited JSON or a length prefix).
        Handle Packet Loss & Reordering: Since UDP is unreliable, design the reader so that it extracts complete JSON messages (using your framing protocol).
    b. UDP Sender (Client)
        Send UDP Packets: Create a function that takes a destination address, marshals a message into JSON, and sends it over UDP.
        Retransmission Strategy: For critical messages (orders, state updates), consider building in a simple ACK mechanism. If an ACK isn’t received within a timeout, retransmit the message.

4. Implement Message Definitions and Parsing
    Define Message Structures:
    In your pkg/message/message.go, define a Go struct that matches your JSON schema. For example:
    Functions to marshal/unmarshal messages.
    Functions to validate message integrity (e.g., checking if the sequence number is in order).

5. Implement the Synchronization Mechanism
    a. Sequence Numbering and ACKs
        Sequence Numbers:
        Each message should include a sequence number to detect gaps. When a receiver notices a jump in the sequence, it knows that some packets were lost.
        Acknowledgments:
        For critical messages (like orders), implement an ACK mechanism. The receiver sends back a simple ACK message containing the sequence number of the received message.
    b. Periodic Full-State Broadcasts
        State Updates:
        To mitigate packet loss over UDP, have the master (or each elevator) periodically broadcast a full state snapshot. This helps resynchronize the nodes even if incremental updates were lost.
        State Reconciliation:
        Implement logic so that when a node detects a missing update, it waits for the next full state message or can explicitly request one from the master.

6. Implement Core Elevator Logic
    a. Master/Slave Coordination
        Master Logic:
        Collect state updates from all elevators.
        Delegate orders to slave nodes.
        Periodically send out full state updates.
        Slave Logic:
        Process orders from the master.
        Send state updates and heartbeats back to the master.
    b. Message Routing and Processing
        Dispatching:
        Create a central function (or use a simple switch-case) that processes incoming messages based on their Type.
        Robust Processing:
        Ensure that repeated or out-of-order messages don’t disrupt the state by designing your update logic to be idempotent.
