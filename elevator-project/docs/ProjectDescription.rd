Project Description

This project is built on a P2P broadcast network, where one elevator acts as a master responsible for delegating hall orders to suitable elevators. 
The master is dynamically elected using heartbeat monitoring, and if it fails, a new one is automatically chosen from the active peers. 
To ensure consistency, elevators broadcast their state regularly, keeping a synchronized world view across the system. 
A recovery system allows elevators to restore their state after failures, 
and a reliable ACK system ensures that messages are received even over an unreliable UDP network.
The system follows a barebone modular architecture, where each module is focused on its core task. 
The APP module acts as the central coordinator, connecting hardware inputs, state machines, and network communication into one unified system. 
Elevators continue serving cab calls during disconnections and automatically reintegrate into the network once reconnected.

APP:
    Coordinates the overall system logic and message passing/handeling.
    It routes incoming network messages, handles button and sensor inputs, manages communication with the elevator FSM, and ensures world view updates are broadcast.
    Includes the central function that connects and synchronizes the elevator logic, hardware, and network layers.
 
CMD:
    Serves as the entry point of the program (main.go).
    Starts network communication, launches the elevator FSM, and begins coordination through the APP. 

CONFIG:
    Holds configurable constants and runtime variables to enable easy tuning of parameters like timeouts, ports, and floor count, without changing code.

DRIVERS:
    Contains the drivers to connect the program to the hardware controlling the elevator.
    Currently includes only the provided hardware interface code.

ELEVATOR:
    Implements the finite state machine (FSM) for each elevator, managing movement, order execution, error detection, and master notifications. 
    Button presses are broadcast as orders until acknowledged by all. If the elevator is the master, it updates the master state. 
    When initialzing new elevator, it attempts to recover its previous state via network queries.

HRA:
    Uses externally provided code to calculate hall order assignments.
    Only the master executes this logic and delegates resulting orders to each elevator.
    Passes new orders to senderWorker in APP to be transmittet on network.

MASTER:
    Handles master election and announcement based on heartbeat monitoring.
    If the current master fails to respond within a timeout, a new master is elected and announced to all elevators. 
    Master election is based on the list of active peers.
    This module ensures there is always a single active master responsible for assigning hall orders and maintaining the global state.

MESSAGE:
    Contains the different message types to be sent on the network. 
    Also includes an ACK tracker system to ensure reliable message delivery in a lossy UDP environment.
    The ACK system enables us to continuously transmitt messages until we are certain the message are received.

NETWORK:
    Responsible for sending and receiving messages over the network. 
    Implements broadcast communication and peer monitoring, including detection of disconnections via heartbeat-based tracking.

REQUESTMATRIX:
    Implements local data structures to store and manage cab and hall request states for each elevator.
    Used by the FSM to determine which orders to serve and when to stop.

SYSTEMDATA:
    Stores the world view of the elevator system.
    Keeps both the local elevator's state and the known states of all other elevators, including their request matrices and availability.       

UTILS:
    Contains general-purpose helper functions used across the project.
    Includes tools for converting types to strings, comparing data structures, and filtering active elevators.