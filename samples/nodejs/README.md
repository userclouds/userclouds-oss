# UserClouds NodeJS Sample App

This sample app demonstrates the core functionality of UserClouds across authentication,
authorization and user store. Follow the instructions below to run the sample app.

This app's error handling is minimized by design to keep the signal-to-noise ratio high
on the sample code. Producing a good user experience with real production usage will require
much more extensive and fine-grained error handling than shown here.

The app requires NPM and Node JS version 18 or higher.

To set up the sample app:

1. Create your UserClouds account. Right now, UserClouds is invite only. If you don't have an account, contact support@userclouds.com
2. Create a new tenant with organizations enabled
3. Create an `.env file` and update with the COMPANY_ID, which is the ID of the only organization listed when opening the Organizations sidebar item
4. Update .env with the TENANT_URL, CLIENT_ID, and CLIENT_SECRET from the default login app, reachable from the "User Authentication" sidebar item in "Login Apps"
5. Update that login app to allow "http://localhost:3000/callback" and "http://localhost:3000/inviteCallback" as REDIRECT_URLs. Remember to click Save.
6. Run the app using the command below, then exit the app. This will create two more organizations.
7. Update .env with the CLIENT_IDs for the two additional login apps
8. Update those additional login apps to also allow "http://localhost:3000/callback" and "http://localhost:3000/inviteCallback" as REDIRECT_URLs, and also make sure they have an Underlying Identity Provider App selected. Again, remember to click Save.
9. Restart the app and create a user account

To run:
`yarn install`
`yarn start`
