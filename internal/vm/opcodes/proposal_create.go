package opcodes

import (
	"github.com/git-town/git-town/v23/internal/forge/forgedomain"
	"github.com/git-town/git-town/v23/internal/git/gitdomain"
	"github.com/git-town/git-town/v23/internal/messages"
	"github.com/git-town/git-town/v23/internal/vm/shared"
	. "github.com/git-town/git-town/v23/pkg/prelude"
)

// ProposalCreate creates a new proposal for the current branch.
type ProposalCreate struct {
	Branch        gitdomain.LocalBranchName
	MainBranch    gitdomain.LocalBranchName
	ProposalBody  Option[gitdomain.ProposalBody]
	ProposalTitle Option[gitdomain.ProposalTitle]
}

func (self *ProposalCreate) Run(args shared.RunArgs) error {
	_, hasParentBranch	:= args.Config.Value.NormalConfig.Lineage.Parent(self.Branch).Get()
	ancestors := args.Config.Value.NormalConfig.Lineage.Ancestors(self.Branch)
	parentBranch := gitdomain.LocalBranchName(ancestors.BranchNames()[0])
	// BranchWith
	if !hasParentBranch {
		args.FinalMessages.Addf(messages.ProposalNoParent, self.Branch)
		return nil
	}
	connector, hasConnector := args.Connector.Get()
	if !hasConnector {
		return forgedomain.UnsupportedServiceError()
	}
	if proposalFinder, canFindProposals := connector.(forgedomain.ProposalFinder); canFindProposals {
		existingProposalOpt, err := proposalFinder.FindProposal(self.Branch, parentBranch)
		if err != nil {
			args.FinalMessages.Addf(messages.ProposalFindProblem, err.Error())
			goto createProposal
		}
		if existingProposal, hasExistingProposal := existingProposalOpt.Get(); hasExistingProposal {
			if args.Config.Value.NormalConfig.BrowserEnabled {
				args.PrependOpcodes(
					&BrowserOpen{
						URL: existingProposal.Data.Data().URL,
					},
				)
			} else {
				args.FinalMessages.Addf(messages.BrowserOpen, existingProposal.Data.Data().URL)
			}
			return nil
		} else {
			// fall back to searching for PRs with the root of the branch hierarchy as the base
			if len(ancestors) > 0 {
				fallbackParent := gitdomain.LocalBranchName(ancestors[0])
				fallbackProposalOpt, err := proposalFinder.FindProposal(self.Branch, fallbackParent)
				if err != nil {
					args.FinalMessages.Addf(messages.ProposalFindProblem, err.Error())
					goto createProposal
				}
				if existingProposal, has := fallbackProposalOpt.Get(); has {
					if args.Config.Value.NormalConfig.BrowserEnabled {
						args.PrependOpcodes(
							&BrowserOpen{
								URL: existingProposal.Data.Data().URL,
							},
						)
					} else {
						args.FinalMessages.Addf(messages.BrowserOpen, existingProposal.Data.Data().URL)
					}
				}
			}
		}
	}

createProposal:
	// TODO: create proposal with embedded lineage here. The lineage is loaded below.
	err := connector.CreateProposal(forgedomain.CreateProposalArgs{
		Branch:         self.Branch,
		FrontendRunner: args.Frontend,
		MainBranch:     self.MainBranch,
		ParentBranch:   parentBranch,
		ProposalBody:   self.ProposalBody,
		ProposalTitle:  self.ProposalTitle,
	})
	if err != nil {
		return err
	}

	if args.Config.Value.NormalConfig.ProposalBreadcrumb.Enabled() {
		// TODO: remove this once we embed the lineage when creating the proposal
		args.PrependOpcodes(&ProposalUpdateBreadcrumb{
			Branch: self.Branch,
		})
	}
	return nil
}
