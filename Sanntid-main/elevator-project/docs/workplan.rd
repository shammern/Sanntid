TODO:
    - Synchronization mechanism
        Goals:  Enable a fault proof, shared and synced world veiw (Sverres remark on our prelim is how this is going to work)
                Sequence number 
                Regular broadcast
                Gap detection
                Ack: Implement Ack on msgType: ButtonEvent, OrderDelegation, ClearOrder
                    -OrderDelegation: Master need to keep track of whick of the elevators that has sent an ack. How shoud this be done? 
                    -ClearOrder: Multiple order might be cleared at the same time (cab + hallup), how can we send and monitor for ack on multiple messages at the same time?
                
    - Master delegation
        Goals: a new master is delegated when the master malfunctions
    - Fault handlig (watchdog, hearbeat, packetloss) -> fourth stage
        Goals: System is robust against packetloss and other malfunctions
                working ack message for hearbeat
    - Errorstate:
        Goals Implement an internal errorstate with appropriate self recovery systems 

BUGS:
    - Elevator always enters DoorOpen state after init. 
    - Prints state = idle twice when entering idlestate

Thoughs: 
   