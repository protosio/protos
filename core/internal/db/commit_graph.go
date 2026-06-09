package db

type CommitGraph struct {
	Items     []CommitGraphItem
	LaneCount int
}

type CommitGraphItem struct {
	Commit      CommitView
	Row         int
	Lane        int
	ActiveLanes []int
	Relations   []CommitGraphRelation
}

type CommitGraphRelation struct {
	ParentHash string
	ParentRow  int
	FromLane   int
	ToLane     int
	Visible    bool
}

func BuildCommitGraph(commits []CommitView) CommitGraph {
	rowByHash := make(map[string]int, len(commits))
	for row, commit := range commits {
		if commit.Hash == "" {
			continue
		}
		if _, exists := rowByHash[commit.Hash]; !exists {
			rowByHash[commit.Hash] = row
		}
	}

	lanes := []string{}
	graph := CommitGraph{
		Items: make([]CommitGraphItem, 0, len(commits)),
	}
	for row, commit := range commits {
		lane := indexOfLane(lanes, commit.Hash)
		if lane < 0 {
			lane = firstEmptyLane(lanes)
			lanes = ensureLane(lanes, lane)
			lanes[lane] = commit.Hash
		}

		activeLanes := activeLaneIndexes(lanes)
		lanes = clearLaneValue(lanes, commit.Hash)

		item := CommitGraphItem{
			Commit:      commit,
			Row:         row,
			Lane:        lane,
			ActiveLanes: activeLanes,
		}
		graph.LaneCount = max(graph.LaneCount, lane+1)

		for parentIndex, parentHash := range uniqueNonEmptyStrings(commit.ParentHashes) {
			parentRow, visible := rowByHash[parentHash]
			if !visible {
				parentRow = -1
			}
			parentLane := indexOfLane(lanes, parentHash)
			if parentLane < 0 && visible {
				if parentIndex == 0 && lane < len(lanes) && lanes[lane] == "" {
					parentLane = lane
				} else {
					parentLane = firstEmptyLane(lanes)
				}
				lanes = ensureLane(lanes, parentLane)
				lanes[parentLane] = parentHash
			}
			if parentLane < 0 {
				parentLane = lane
			}
			item.Relations = append(item.Relations, CommitGraphRelation{
				ParentHash: parentHash,
				ParentRow:  parentRow,
				FromLane:   lane,
				ToLane:     parentLane,
				Visible:    visible,
			})
			graph.LaneCount = max(graph.LaneCount, parentLane+1)
		}

		lanes = trimTrailingEmptyLanes(lanes)
		graph.LaneCount = max(graph.LaneCount, len(lanes))
		graph.Items = append(graph.Items, item)
	}

	return graph
}

func indexOfLane(lanes []string, value string) int {
	if value == "" {
		return -1
	}
	for lane, laneValue := range lanes {
		if laneValue == value {
			return lane
		}
	}
	return -1
}

func firstEmptyLane(lanes []string) int {
	for lane, value := range lanes {
		if value == "" {
			return lane
		}
	}
	return len(lanes)
}

func ensureLane(lanes []string, lane int) []string {
	for len(lanes) <= lane {
		lanes = append(lanes, "")
	}
	return lanes
}

func clearLaneValue(lanes []string, value string) []string {
	if value == "" {
		return lanes
	}
	for lane, laneValue := range lanes {
		if laneValue == value {
			lanes[lane] = ""
		}
	}
	return lanes
}

func activeLaneIndexes(lanes []string) []int {
	active := make([]int, 0, len(lanes))
	for lane, value := range lanes {
		if value != "" {
			active = append(active, lane)
		}
	}
	return active
}

func trimTrailingEmptyLanes(lanes []string) []string {
	for len(lanes) > 0 && lanes[len(lanes)-1] == "" {
		lanes = lanes[:len(lanes)-1]
	}
	return lanes
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
