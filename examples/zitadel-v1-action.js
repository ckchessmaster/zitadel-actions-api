/**
 * ZITADEL Actions V1 JavaScript Script (Fallback)
 *
 * Flow: Complement Token -> Pre access token creation (or Pre userinfo creation)
 *
 * This action calls the zitadel-actions-api service to extract user roles and flatten
 * them into a 'groups' claim for oauth2-proxy / Frigate.
 */

const http = require('zitadel/http');

function complementTokenWithGroups(ctx, api) {
    // API endpoint (internal cluster address or exposed ingress)
    const endpoint = 'http://zitadel-actions-api.default.svc.cluster.local/v1/actions/flatten-roles';

    // Build payload containing user grants and claims
    const payload = JSON.stringify({
        userID: ctx.v1.user.id,
        user_grants: ctx.v1.user.grants ? ctx.v1.user.grants.grants : [],
        claims: ctx.v1.claims
    });

    try {
        const response = http.post(endpoint, {
            headers: {
                'Content-Type': 'application/json'
            },
            body: payload
        });

        if (response.status === 200) {
            const data = JSON.parse(response.body);
            if (data.groups && Array.isArray(data.groups) && data.groups.length > 0) {
                // Set the flattened groups claim in the token
                api.v1.claims.setClaim('groups', data.groups);
            }
        }
    } catch (e) {
        // Fall open or log error to prevent breaking login flow
        // api.v1.claims.setClaim('error_action', e.toString());
    }
}
