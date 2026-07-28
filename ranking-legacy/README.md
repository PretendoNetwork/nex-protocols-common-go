# Ranking (legacy) implementation

Docs: [Nintendo-wiki](https://nintendo-wiki.pretendo.network/docs/nex/protocols/ranking/legacy)

Current implementation notes:
- Users are identified not by PID, but by (pid, unique_id). Unique IDs are issued by storage-manager to allow multiple "profiles" per account. So, everywhere a PID is passed, expect a unique ID as well.
- Similarly, (pid, unique_id, category) identifies one specific score.
- The Limit parameter in UploadScoreWithLimit and friends is not documented or understood, so is discarded. Games *may* need to undergo leaderboard resets once this is worked out, depending on the details.
- (Limit might be related to expiry time? "Weekly rankings"?)
- There's two unknown parameters to most rankings. These are presumably some kind of extra attributes or category that can be filtered on, but this is not implemented. The values *are* saved and returned to clients.
- Unique IDs are *not* checked against storage-manager to ensure they were actually issued.
- Games can use the full range of uint32 or uint64 - please be careful not to use signed types that are too small, especially in Postgres!

todo for merge:
- NEX 1 CommonData allocation behaviour not implemented yet
- audit result codes, method rmc IDs, structure of return values
- database indexes
- some kind of handling for RankingMode (friends)