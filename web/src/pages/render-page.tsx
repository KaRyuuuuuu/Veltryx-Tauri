import type { PageId, WorkspacePageProps } from "../types/app";
import AccountPage from "./account";
import ViewerHomePage from "./viewer/home";
import ViewerLivePage from "./viewer/live";
import ViewerMapPage from "./viewer/map";
import ViewerSectorsPage from "./viewer/sectors";
import ViewerFlagsPage from "./viewer/flags";
import ViewerReportsPage from "./viewer/reports";
import RaceControlOverviewPage from "./race-control/overview";
import RaceControlFlagsPage from "./race-control/flags";
import RaceControlRacelinksPage from "./race-control/racelinks";
import RaceControlTmsPage from "./race-control/tms";
import RaceControlBaselinksPage from "./race-control/baselinks";
import BridgePage from "./bridge";
import BridgeHealthPage from "./bridge/health";
import BridgeLogsPage from "./bridge/logs";
import AdminConnectedPage from "./admin/connected";
import AdminAccountsPage from "./admin/accounts";
import AdminUsersPage from "./admin/users";
import AdminEventsPage from "./admin/events";
import AdminSettingsPage from "./admin/settings";

type RenderPageProps = WorkspacePageProps & {
  page: PageId;
};

export function RenderPageContent({ page, ...props }: RenderPageProps) {
  switch (page) {
    case "account-profile":
      return <AccountPage {...props} />;
    case "viewer-home":
      return <ViewerHomePage {...props} />;
    case "viewer-live":
      return <ViewerLivePage {...props} />;
    case "viewer-map":
      return <ViewerMapPage {...props} />;
    case "viewer-sectors":
      return <ViewerSectorsPage {...props} />;
    case "viewer-flags":
      return <ViewerFlagsPage {...props} />;
    case "viewer-reports":
      return <ViewerReportsPage {...props} />;
    case "rc-overview":
      return <RaceControlOverviewPage {...props} />;
    case "rc-flags":
      return <RaceControlFlagsPage {...props} />;
    case "rc-racelinks":
      return <RaceControlRacelinksPage {...props} />;
    case "rc-tms":
      return <RaceControlTmsPage {...props} />;
    case "rc-baselinks":
      return <RaceControlBaselinksPage {...props} />;
    case "bridge":
      return <BridgePage {...props} />;
    case "bridge-health":
      return <BridgeHealthPage {...props} />;
    case "logs":
      return <BridgeLogsPage {...props} />;
    case "admin-presence":
      return <AdminConnectedPage {...props} />;
    case "admin-accounts":
      return <AdminAccountsPage {...props} />;
    case "admin-users":
      return <AdminUsersPage {...props} />;
    case "admin-events":
      return <AdminEventsPage {...props} />;
    case "admin-settings":
      return <AdminSettingsPage {...props} />;
    default:
      return null;
  }
}
