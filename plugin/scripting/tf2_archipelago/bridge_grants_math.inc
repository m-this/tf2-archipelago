/**
 * State-grant progress independent of the effect acknowledgement cursor.
 * This include has no natives so the SourcePawn VM can run the same decision
 * the live grant callback uses when a held effect causes repeated polls.
 */

int Bridge_NextStateGrantSeq(int seen, int received)
{
    return received > seen ? received : seen;
}
