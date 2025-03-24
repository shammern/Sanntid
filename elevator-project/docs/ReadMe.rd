How to run program:
step 1:
    -Start simelevator simelevservers
    -(on windows) CD to where .exe is located
    simelevatorserver --port 15555
    simelevatorserver --port 15556
    simelevatorserver --port 15557
    chmod a+rwx hall_request_assigner

    ./SimElevatorServer --port 15555

step 2: 
    -Run go mains
    -CD to projectfile/cmd/main
    go run main.go --id=1
    go run main.go --id=2
    go run main.go --id=3


How to enable packetloss:

    sudo netimpair -f
    sudo netimpair --off
    sudo netimpair --preset off
        Disables all network impairments
        
    sudo netimpair -p 12345,23456,34567 --loss 25
        Applies 25%% packet loss to ports 12345, 23456, and 34567
        
    sudo netimpair -n executablename --preset medium
        Applies the 'medium' impairment preset to all ports used by all programs named 'executablename'
        The program will keep running in order to continuously monitor what ports 'executablename' uses
        (for elixir programs, executable name would typically be 'beam.smp')
        
    sudo netimpair -n executablename -f
        Continuously monitors ports used by 'executablename', but does not apply any impariments
