# student_17_bftcosi
Consensus omniledger-like
## Description
The purpose of this work is to implement a robust and scalable consensus algorithm using CoSi protocol and handling some exceptions. **The CoSi tree is a three level tree** to make a compromise between the two-level tree, making the root-node vulnerable to DoS, and a more than three level tree, slowing the algorithm because of the RTT between the root node and the leaves.

The tree is composed of a leader (root-node), and some groups of equal size, each having a sub-leader (second level nodes) and members (leaves). The groups composition are defined by the leader.

We want to **handle non-responding nodes**, no matter where they are in the tree. If a leaf is failing, then it is ignored in the CoSi commitment. If a sub-leader is non-responding, then the leader (root node) recreates the group designing another sub-leader from the group members. And finally, if the leader is failing, the protocol restarts using another leader.

More complex adversaries (modifying messages, non-responding at challenge time, etc.) are not yet handled.
The purpose of the project is to **test scalability and robustness** of this service on a testbed and to have a well-documented **reusable code** for it.


## References
- OmniLedger: A Secure, Scale-Out, Decentralized Ledger via Sharding: https://eprint.iacr.org/2017/406.pdf part 4 A & B
- (CoSi) Keeping Authorities "Honest or Bust" with Decentralized Witness Cosigning: https://arxiv.org/abs/1503.08768
- (ByzCoin) Enhancing Bitcoin Security and Performance with Strong Consistency via Collective Signing: https://arxiv.org/abs/1602.06997
