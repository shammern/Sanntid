# Sanntid
TTK4145 - Sanntid

This project is base on a P2P multicast network where one elevator is working as a master delegating orders to a suitable elevator. 
Master and backup elevator is based on a simple predefined ruleset. Heartbeat monitoring is used as an alive signal. Missing heatbeats will eventualle be handled as a disconnection.
If one of the other elevators stops receiving heartbeats from the master a new master will be delegated and broadcasted on the network. If the backup disconnects the master will
delegate a new backup to ensure that a backup is always active. The world views will be transmitted at regular intervals to ensure a syncronised world view. 
A system to handle differences will be implemated. The main logic to connect the different modules and handle logic to run the program is found in the APP package.
Therefore the other modules will have a barebone approach with the goal to just have the needed skeleton code. 



APP:
    App handles all program related work. It connect the barebone functionality made in other modules with logic to make it suitable to run an elevator. 
    E.g it connects the transports layers receive function with logic to handle the incomming message and store the information received. 

CONFIG:
    Is thus far some simple maps to store basic config parameters to enable a uncomplicated setup of the project.

DRIVERS:
    Contains the drivers to connect the program to the hardware controlling the elevator. Currently only contains handed out code.

ELEVATOR:
    This is where the statemachine for each elevator is located. It still has some remains of single elevator code as it is still used for debugging and partial implementation.

MESSAGE:
    Contains the different message types to be sent on the network. As well as functions to prepare the messages for transmittion (marshalling to JSON)

ORDER:
    This is where the request matrix is created. This is also where the logic to assign new orders to a suitable elevator is located. 

STATE:
    This package stores the world view of the elevator. It stores both it's own state and request matrix as well as the same for the other elevators. 

TRANSPORT:
    This out network module where we handle both sending and receiving messages over UDP multicast.

MASTER:
    This contains a snippet of the code to delegate active master based on heatbeat monitoring. This is a early implementation and has not been incorperated with the project. 
    It is however included to give insight into our initial plan. 