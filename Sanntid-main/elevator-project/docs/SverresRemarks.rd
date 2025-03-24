Hi Group x,


OK, you have been thinking. That is great :-) There are more
ingredients that necessary to solve the problem here it seems to me
(and there are some misunderstandings also...), but let me reflect
back to you what I get from the report:

All nodes broadcast regularly its world-view. Great.

Is this enough to make all nodes 'synchronized'? Not really, and you
do talk about a 'version control system' that should help with this. I
do not get enough information about this. Anyway since you also have a
master this is no problem: The only world view that counts is the
master's.

The master will delegate orders to elevators. (I must assume it
decides to turn on and off lamps, and removes orders from the
'official' system state when appropiate?) 

Hmm. Now there are two crucial questions:

 * Are the state synchronized 'enough' that the master can be killed
   at any time without any orders being lost? -> lamp doesn't get lit before all elevators have acknoleged the order. 
   Might ad a delay if the other elevators doesn't ack the order in reasnable time. Maybe disregard elevators that are offline? 
   Threshold will need to be found to ensure a balanced performance. Use libp2p's built in error handlig will solve alot of these problems.

 * Are you making the system more complicated for yourselves than
   necessary? -> p2p complexity? doesn't need a dedicated backup, but more complex yes.

The answer to the second is 'yes' I think. And to the first... I need
more information but check this: 

Assume a master node button is pressed. It will be broadcasted, but
the other nodes do not get it.  When a master hears about the order it
will delegate it. Assume it delegates it to itself, and this message
neither reach anybody else. The lamp will now be turned on. The master
crashes, and nobody knows about the order with the lamp on.

You are 'thinking in the right manner' about the problem. Your
ingredients are good, even though I lack details. But have lost a bit
of the big picture? Have you gotten into 'brainstorm mode' and lost
focus of the problem you are solving?

Sverre