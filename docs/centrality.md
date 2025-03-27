## Centrality

Hydrus calculates 4 different centrality metrics to determine how cardinal is a node in the network: degree, betweenness, closeness and eigenvector.

### Degree

Degree centrality is a measure to determine the importance of a node within a graph. 

It quantifies how connected a node is by counting the number of channels it has to other nodes. A higher degree centrality indicates that a node is more central or influential within the network.

> In a social network graph, if node A has 10 channels, and node B has 2, then node A would have a higher degree centrality than node B, making it more central in the network.

### Betweenness

Betweenness centrality is a measure to determine the importance of a node based on its role in connecting different parts of the network.

Unlike degree centrality, which counts the number of direct connections a node has, betweenness centrality focuses on how often a node lies on the shortest paths between pairs of other nodes.

> Nodes with high betweenness centrality act as bridges, connecting different parts of the network.

### Closeness

Closeness centrality quantifies how centrally located a node is within a graph by measuring its average proximity to all other nodes.

> Closeness centrality for a node *A* is the reciprocal of the sum of the shortest path distances from *A* to all other reachable nodes in the network.

### Eigenvector

The Eigenvector centrality is a measure to determine the influence or importance of nodes within a network, the score is calculated based on the principle that if one node has a high centrality, then all of its neighbors should also have high centralities.
