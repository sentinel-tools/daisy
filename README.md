# Overview 

Daisy is a CLI tool for managing Redis daisy-chained slave pools.

Imagine you have a basic Redis pod of a master and a pair of slaves. Now
imagine you want to add an additional pool of slaves which are slaved to some
combinaton of the main pod members, and you don't want these additional slaves
to be eligible for failover promotion. In other words you want a pool of
read-only slaves.

You could manage it the manual way, or you could let Daisy configure it the way
you want it.

# Pool Policies

There are multiple criteria you could use in determining what pool nodes slave
to which primary pool. Daisy refers to these as policies.  In Daisy we call the
slaves in the pod the "primary pool" and the secondary (chained) slaves the
"slave pool".  By default Daisy will query Sentinel for the slaves of the given
pod.

## Random

Under this policy the secondary pool nodes are enslaved to random nodes in the
primary pool.

## One For One

Under this policy every secondary pool node is assigned one primary pool node.
For this to work the number of primary pool must be equal to or greater than
the number of secondary pool nodes. For example if you have a 4 node read-pool
you need four slave nodes in the pod.

## Ring Pool

This is similar to One For One, except it treats the primary pool as slots and
hashes an index to assign a secondary pool node, assigning them in sequence.

For example say you have three primary pool, and want six secopndary pool
nodes. Daisy will iterate over the primary pool, looping back around when it
reaches the last one - hence operating in a "ring". In this example it will
result in each pod node having a pair of slaves in the pool.
