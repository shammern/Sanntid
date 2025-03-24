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
    - Fault handlig (watchdog, packetloss) -> fourth stage
        Goals: System is robust against packetloss and other malfunctions
                working ack message for hearbeat
    

BUGS:
    - 

Thoughs: 


Plan:
    - Check packetloss: Sigve
    - Master doen't change unneccesary: Selma
    - Internal elevator errorstate: Door open too long, stoppes betwhen floors, too long time betwhen floors etc
    - If an elevator enters errorstate (powerloss etc) it need's to get caborders back when it exits errorstate: Wei

   
