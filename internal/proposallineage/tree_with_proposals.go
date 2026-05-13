package proposallineage

import (
	"github.com/git-town/git-town/v23/internal/config/configdomain"
	"github.com/git-town/git-town/v23/internal/forge/forgedomain"
	"github.com/git-town/git-town/v23/internal/git/gitdomain"
	. "github.com/git-town/git-town/v23/pkg/prelude"
)

type TreeNodeWithProposal struct {
	Branch   gitdomain.LocalBranchName
	Children []TreeNodeWithProposal
	Proposal Option[forgedomain.Proposal]
}

func (self TreeNodeWithProposal) BranchOrAncestorHasProposal() bool {
	if self.Proposal.IsSome() {
		return true
	}
	for _, child := range self.Children {
		if child.BranchOrAncestorHasProposal() {
			return true
		}
	}
	return false
}

func AddProposalsToTree(lineage configdomain.Lineage, tree TreeNode, connector Option[forgedomain.Connector]) TreeNodeWithProposal {
	return addProposalsToTreeHelper(lineage, tree, None[gitdomain.LocalBranchName](), connector)
}

func addProposalsToTreeHelper(lineage configdomain.Lineage, tree TreeNode, parent Option[gitdomain.LocalBranchName], connector Option[forgedomain.Connector]) TreeNodeWithProposal {
	proposal := loadProposal(tree.Branch, parent, connector)
	if proposal.IsNone() {
		ancestors := lineage.Ancestors(tree.Branch)
		proposal = loadProposal(tree.Branch, Some(ancestors[0]), connector)
	}
	children := make([]TreeNodeWithProposal, len(tree.Children))
	for i, child := range tree.Children {
		children[i] = addProposalsToTreeHelper(lineage, child, Some(tree.Branch), connector)
	}
	return TreeNodeWithProposal{
		Branch:   tree.Branch,
		Children: children,
		Proposal: proposal,
	}
}

func loadProposal(branch gitdomain.LocalBranchName, parentOpt Option[gitdomain.LocalBranchName], connectorOpt Option[forgedomain.Connector]) Option[forgedomain.Proposal] {
	parent, hasParent := parentOpt.Get()
	connector, hasConnector := connectorOpt.Get()
	if !hasParent || !hasConnector {
		return None[forgedomain.Proposal]()
	}
	finder, canFindProposals := connector.(forgedomain.ProposalFinder)
	if !canFindProposals {
		return None[forgedomain.Proposal]()
	}
	proposal, err := finder.FindProposal(branch, parent)
	if err != nil {
		return None[forgedomain.Proposal]()
	}
	return proposal
}
