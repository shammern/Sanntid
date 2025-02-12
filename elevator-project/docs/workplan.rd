Workplan:
    - Network Module / communication framework -> first stage
        Goals:  get three elevators connected together using UDP, both receiving and sending should be possible
                P2P network working and stable
    - Synchronization mechanism -> second stage
        Goals:  Enable a fault proof, shared and synced world veiw (Sverres remark on our prelim is how this is going to work)
                Have a working JSON message for worldview and other messagetypes
                Sequence number 
                Regular broadcast
                Gap detection
                
    - Cost function / order delegation between elevators -> second stage
        Goals: properly distribute orders to the optimal elevator
    - Master delegation -> third stage
        Goals: a new master is delegated when the master malfunctions
    - Fault handlig (watchdog, hearbeat, packetloss) -> fourth stage
        Goals: System is robust against packetloss and other malfunctions
                working ack message for hearbeat
