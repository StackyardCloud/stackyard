package server

import "fmt"

// Channel memberships, moderators and bans.

// addMember adds who to ch (or returns its existing membership). Elastic
// channels place the member in sub (when given) or in the first sub-channel
// below its target size, opening sub-channels up to the configured maximum.
func (st *cmsgStore) addMember(ch *cmsgChannel, who, typ, inviter, sub string) (*cmsgMember, *chimeSDKErr) {
	if ch.bans[who] != nil {
		return nil, chimeSDKForbidden("%s is banned from channel %s.", who, ch.arn)
	}
	if m := ch.members[who]; m != nil {
		return m, nil
	}
	if ch.elastic == nil && len(ch.members) >= cmsgMaxMembers {
		return nil, chimeSDKLimit("A channel can have at most %d members.", cmsgMaxMembers)
	}
	if st.memberships >= cmsgMaxMemberships {
		return nil, chimeSDKLimit("The account can have at most %d channel memberships.", cmsgMaxMemberships)
	}
	if ch.elastic != nil {
		var cerr *chimeSDKErr
		if sub, cerr = ch.placeInSubChannel(sub); cerr != nil {
			return nil, cerr
		}
	} else if sub != "" {
		return nil, chimeSDKBadRequest("SubChannelId applies only to elastic channels.")
	}
	now := st.now()
	m := &cmsgMember{arn: who, typ: cmsgDefault(typ, "DEFAULT"), invitedBy: inviter, sub: sub, created: now, updated: now}
	ch.members[who] = m
	if sub != "" {
		ch.subChannels[sub]++
	}
	chimeSDKPut(st.userChannels, who, ch.arn)
	st.memberships++
	return m, nil
}

func (ch *cmsgChannel) placeInSubChannel(sub string) (string, *chimeSDKErr) {
	target := mmgrInt(ch.elastic, "TargetMembershipsPerSubChannel", 2)
	if sub != "" {
		count, ok := ch.subChannels[sub]
		switch {
		case !ok:
			return "", chimeSDKNotFound("Sub-channel %s doesn't exist.", sub)
		case count >= target:
			return "", chimeSDKLimit("Sub-channel %s is full.", sub)
		}
		return sub, nil
	}
	for id, count := range ch.subChannels {
		if count < target {
			return id, nil
		}
	}
	if len(ch.subChannels) >= mmgrInt(ch.elastic, "MaximumSubChannels", 2) {
		return "", chimeSDKLimit("Elastic channel %s has no room for more members.", ch.arn)
	}
	ch.subSeq++
	id := fmt.Sprintf("sub-%05d", ch.subSeq)
	ch.subChannels[id] = 0
	return id, nil
}

func (st *cmsgStore) removeMember(ch *cmsgChannel, who string) {
	m := ch.members[who]
	if m == nil {
		return
	}
	if m.sub != "" {
		ch.subChannels[m.sub]--
	}
	delete(ch.members, who)
	chimeSDKDrop(st.userChannels, who, ch.arn)
	st.memberships--
}

func (st *cmsgStore) addModerator(ch *cmsgChannel, who, by string) *chimeSDKErr {
	if ch.moderators[who] != nil {
		return nil
	}
	if len(ch.moderators) >= cmsgMaxModerators {
		return chimeSDKLimit("A channel can have at most %d moderators.", cmsgMaxModerators)
	}
	ch.moderators[who] = &cmsgRole{createdBy: by, created: st.now()}
	chimeSDKPut(st.userModerated, who, ch.arn)
	return nil
}

// canAdd reports whether bearer may add member: moderators always, members
// of unrestricted channels, and anyone joining a public channel themselves.
func (ch *cmsgChannel) canAdd(bearer, member string) *chimeSDKErr {
	if ch.isModerator(bearer) || (ch.mode == "UNRESTRICTED" && ch.members[bearer] != nil) || (ch.privacy == "PUBLIC" && member == bearer) {
		return nil
	}
	return chimeSDKForbidden("%s can't add members to channel %s.", bearer, ch.arn)
}

func (ch *cmsgChannel) membershipView(m *cmsgMember) map[string]any {
	out := map[string]any{
		"InvitedBy": cmsgIdentityView(m.invitedBy), "Type": m.typ, "Member": cmsgIdentityView(m.arn), "ChannelArn": ch.arn,
		"CreatedTimestamp": chimeSDKTime(m.created), "LastUpdatedTimestamp": chimeSDKTime(m.updated),
	}
	if m.sub != "" {
		out["SubChannelId"] = m.sub
	}
	return out
}

func (st *cmsgStore) createMembership(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	who := mmgrStr(in, "MemberArn")
	if cerr := cmsgMemberARN(who, ch.appArn); cerr != nil {
		return nil, cerr
	}
	if cerr := ch.canAdd(bearer, who); cerr != nil {
		return nil, cerr
	}
	m, cerr := st.addMember(ch, who, mmgrStr(in, "Type"), bearer, mmgrStr(in, "SubChannelId"))
	if cerr != nil {
		return nil, cerr
	}
	out := map[string]any{"ChannelArn": ch.arn, "Member": cmsgIdentityView(who)}
	if m.sub != "" {
		out["SubChannelId"] = m.sub
	}
	return out, nil
}

func (st *cmsgStore) batchCreateMembership(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	typ := cmsgDefault(mmgrStr(in, "Type"), "DEFAULT")
	members, errs := []any{}, []any{}
	for _, raw := range mmgrList(in, "MemberArns") {
		who := mmgrStrOf(raw)
		cerr := cmsgMemberARN(who, ch.appArn)
		if cerr == nil {
			cerr = ch.canAdd(bearer, who)
		}
		if cerr == nil {
			_, cerr = st.addMember(ch, who, typ, bearer, mmgrStr(in, "SubChannelId"))
		}
		if cerr != nil {
			errs = append(errs, map[string]any{"MemberArn": who, "ErrorCode": chimeSDKErrorCodes[cerr.code], "ErrorMessage": cerr.msg})
			continue
		}
		members = append(members, cmsgIdentityView(who))
	}
	batch := map[string]any{"InvitedBy": cmsgIdentityView(bearer), "Type": typ, "Members": members, "ChannelArn": ch.arn}
	if sub := mmgrStr(in, "SubChannelId"); sub != "" {
		batch["SubChannelId"] = sub
	}
	return map[string]any{"BatchChannelMemberships": batch, "Errors": errs}, nil
}

func (st *cmsgStore) member(ch *cmsgChannel, who string) (*cmsgMember, *chimeSDKErr) {
	m := ch.members[who]
	if m == nil {
		return nil, chimeSDKNotFound("%s isn't a member of channel %s.", who, ch.arn)
	}
	return m, nil
}

func (st *cmsgStore) describeMembership(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.readable(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	m, cerr := st.member(ch, mmgrStr(in, "MemberArn"))
	if cerr != nil {
		return nil, cerr
	}
	if sub := mmgrStr(in, "SubChannelId"); sub != "" && sub != m.sub {
		return nil, chimeSDKNotFound("%s isn't a member of sub-channel %s.", m.arn, sub)
	}
	return map[string]any{"ChannelMembership": ch.membershipView(m)}, nil
}

func (st *cmsgStore) deleteMembership(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, cerr
	}
	who := mmgrStr(in, "MemberArn")
	if who != bearer && !ch.isModerator(bearer) {
		return nil, chimeSDKForbidden("Only moderators can remove other members from channel %s.", ch.arn)
	}
	if _, cerr := st.member(ch, who); cerr != nil {
		return nil, cerr
	}
	st.removeMember(ch, who)
	return nil, nil
}

func (st *cmsgStore) listMemberships(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.readable(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	typ, sub := cmsgDefault(mmgrStr(in, "Type"), "DEFAULT"), mmgrStr(in, "SubChannelId")
	var arns []string
	for who, m := range ch.members {
		if m.typ == typ && (sub == "" || m.sub == sub) {
			arns = append(arns, who)
		}
	}
	page, next := chimeSDKPage(arns, mmgrStr(in, "NextToken"), mmgrInt(in, "MaxResults", 0), 50)
	out := make([]any, 0, len(page))
	for _, who := range page {
		out = append(out, map[string]any{"Member": cmsgIdentityView(who)})
	}
	resp := map[string]any{"ChannelArn": ch.arn, "ChannelMemberships": out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

// ownMembership returns the bearer's own membership named by MemberArn.
func (st *cmsgStore) ownMembership(in map[string]any, bearer string) (*cmsgChannel, *cmsgMember, *chimeSDKErr) {
	ch, cerr := st.channel(mmgrStr(in, "ChannelArn"), bearer)
	if cerr != nil {
		return nil, nil, cerr
	}
	who := mmgrStr(in, "MemberArn")
	if who != bearer {
		return nil, nil, chimeSDKForbidden("Members can only manage their own preferences.")
	}
	m, cerr := st.member(ch, who)
	return ch, m, cerr
}

func (st *cmsgStore) getPreferences(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, m, cerr := st.ownMembership(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	out := map[string]any{"ChannelArn": ch.arn, "Member": cmsgIdentityView(m.arn)}
	if m.prefs != nil {
		out["Preferences"] = chimeSDKCopy(m.prefs)
	}
	return out, nil
}

func (st *cmsgStore) putPreferences(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, m, cerr := st.ownMembership(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	prefs := mmgrMap(in, "Preferences")
	push := mmgrMap(prefs, "PushNotifications")
	if filtered := mmgrStr(push, "AllowNotifications") == "FILTERED"; filtered != (mmgrStr(push, "FilterRule") != "") {
		return nil, chimeSDKBadRequest("FilterRule is required with, and only with, AllowNotifications FILTERED.")
	}
	m.prefs = prefs
	m.updated = st.now()
	return map[string]any{"ChannelArn": ch.arn, "Member": cmsgIdentityView(m.arn), "Preferences": chimeSDKCopy(prefs)}, nil
}

// ---- moderators ----

func cmsgRoleView(ch *cmsgChannel, key, who string, role *cmsgRole) map[string]any {
	return map[string]any{key: cmsgIdentityView(who), "ChannelArn": ch.arn, "CreatedTimestamp": chimeSDKTime(role.created), "CreatedBy": cmsgIdentityView(role.createdBy)}
}

func (st *cmsgStore) createModerator(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	who := mmgrStr(in, "ChannelModeratorArn")
	if cerr := cmsgMemberARN(who, ch.appArn); cerr != nil {
		return nil, cerr
	}
	if ch.bans[who] != nil {
		return nil, chimeSDKForbidden("%s is banned from channel %s.", who, ch.arn)
	}
	if cerr := st.addModerator(ch, who, bearer); cerr != nil {
		return nil, cerr
	}
	return map[string]any{"ChannelArn": ch.arn, "ChannelModerator": cmsgIdentityView(who)}, nil
}

func (st *cmsgStore) describeModerator(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.readable(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	who := mmgrStr(in, "ChannelModeratorArn")
	role := ch.moderators[who]
	if role == nil {
		return nil, chimeSDKNotFound("%s doesn't moderate channel %s.", who, ch.arn)
	}
	return map[string]any{"ChannelModerator": cmsgRoleView(ch, "Moderator", who, role)}, nil
}

func (st *cmsgStore) deleteModerator(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	who := mmgrStr(in, "ChannelModeratorArn")
	if ch.moderators[who] == nil {
		return nil, chimeSDKNotFound("%s doesn't moderate channel %s.", who, ch.arn)
	}
	delete(ch.moderators, who)
	chimeSDKDrop(st.userModerated, who, ch.arn)
	return nil, nil
}

func cmsgPageRoles(ch *cmsgChannel, roles map[string]*cmsgRole, in map[string]any, listKey, itemKey string) map[string]any {
	arns := make([]string, 0, len(roles))
	for who := range roles {
		arns = append(arns, who)
	}
	page, next := chimeSDKPage(arns, mmgrStr(in, "NextToken"), mmgrInt(in, "MaxResults", 0), 50)
	out := make([]any, 0, len(page))
	for _, who := range page {
		out = append(out, map[string]any{itemKey: cmsgIdentityView(who)})
	}
	resp := map[string]any{"ChannelArn": ch.arn, listKey: out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp
}

func (st *cmsgStore) listModerators(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.readable(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	return cmsgPageRoles(ch, ch.moderators, in, "ChannelModerators", "Moderator"), nil
}

// ---- bans ----

func (st *cmsgStore) createBan(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	who := mmgrStr(in, "MemberArn")
	if cerr := cmsgMemberARN(who, ch.appArn); cerr != nil {
		return nil, cerr
	}
	if ch.isModerator(who) {
		return nil, chimeSDKBadRequest("Moderators can't be banned; remove %s as a moderator first.", who)
	}
	if ch.bans[who] == nil {
		if len(ch.bans) >= cmsgMaxBans {
			return nil, chimeSDKLimit("A channel can have at most %d bans.", cmsgMaxBans)
		}
		ch.bans[who] = &cmsgRole{createdBy: bearer, created: st.now()}
	}
	// A ban removes the member from the channel.
	st.removeMember(ch, who)
	return map[string]any{"ChannelArn": ch.arn, "Member": cmsgIdentityView(who)}, nil
}

func (st *cmsgStore) describeBan(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	who := mmgrStr(in, "MemberArn")
	ban := ch.bans[who]
	if ban == nil {
		return nil, chimeSDKNotFound("%s isn't banned from channel %s.", who, ch.arn)
	}
	return map[string]any{"ChannelBan": cmsgRoleView(ch, "Member", who, ban)}, nil
}

func (st *cmsgStore) deleteBan(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	who := mmgrStr(in, "MemberArn")
	if ch.bans[who] == nil {
		return nil, chimeSDKNotFound("%s isn't banned from channel %s.", who, ch.arn)
	}
	delete(ch.bans, who)
	return nil, nil
}

func (st *cmsgStore) listBans(in map[string]any, bearer string) (map[string]any, *chimeSDKErr) {
	ch, cerr := st.moderated(in, bearer)
	if cerr != nil {
		return nil, cerr
	}
	return cmsgPageRoles(ch, ch.bans, in, "ChannelBans", "Member"), nil
}
