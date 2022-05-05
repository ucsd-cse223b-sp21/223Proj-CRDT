# 223Proj-CRDT
RGA-based CRDT Text-Editor

We are interested in exploring peer-to-peer collaborative editing using Conflict-free Replicated Data Types (CRDTs). To that end, we provide a lightweight peerto-peer collaborative editing tool that can offer a reliable local editing experience even in the absence of a data connection and meeting the CCI model of consistency. We provide a software artifact consisting of a text editing interface that allows reads and appends/removes from a ”document” (string-like data structure) and an RGA-based CRDT which passes along updates to the document from the user to the network and vice versa. We optimized our system to achieve both fast local and downstream operations (sustained latency of upstream/downstream operations are proportional) and maintain memory
usage linear in the size of the document. Evaluation consists of ensuring that these performance guarantees along with those of the CCI model are met under a relatively fast network, but where network disconnections of a peer may occur at any time and for some parameterizable length of time. 

Link to the full paper: https://drive.google.com/file/d/1GFmvMzUosztbu0kUa0OBLlhaHgR6lYrX/view?usp=sharing
