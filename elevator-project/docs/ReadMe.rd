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

