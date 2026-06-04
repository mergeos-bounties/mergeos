import React from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import Header from '../components/Header';
import Sidebar from '../components/Sidebar';
import './DashboardLayout.css';

/**
 * DashboardLayout provides the main structure for authenticated views, 
 * including a header, sidebar, and a main content area.
 *
 * This component resolves a bug where project detail pages, when navigated to 
 * from the dashboard, would inherit restrictive layout styles (e.g., max-width)
 * intended only for dashboard-level views. This caused visual bugs like content 
 * clipping, overflow, and incorrect spacing.
 *
 * The fix is to make the layout context-aware. By using the `useLocation` hook
 * from react-router-dom, the component checks if the current path corresponds to a 
 * project detail view. If it does, it applies a 'full-width' CSS class to the main
 * content wrapper, removing the problematic constraints and allowing the project
 * view to render correctly with its own intended layout.
 */
const DashboardLayout: React.FC = () => {
  const location = useLocation();

  // Determine if the current view is a project detail page.
  // This is the core of the fix: dynamically changing layout based on route.
  const isProjectView = location.pathname.includes('/project/');

  const contentWrapperClassName = `content-wrapper ${
    isProjectView ? 'content-wrapper--full-width' : ''
  }`.trim();

  return (
    <div className="dashboard-layout">
      <Header />
      <div className="dashboard-main-container">
        <Sidebar />
        <main className={contentWrapperClassName}>
          <Outlet />
        </main>
      </div>
    </div>
  );
};

export default DashboardLayout;

/*
================================================================================
 ASSOCIATED CSS (`src/layouts/DashboardLayout.css`)
 This is an illustration of the required CSS changes.
================================================================================

.dashboard-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: #f4f7fa;
}

.dashboard-main-container {
  display: flex;
  flex: 1;
  overflow: hidden; 
}

.content-wrapper {
  flex: 1;
  padding: 1.5rem 2rem;
  overflow-y: auto;
  
  // PROBLEM: These styles constrain the project view, causing layout bugs.
  max-width: 1280px;
  margin-left: auto;
  margin-right: auto;
}

// FIX: This modifier class is conditionally applied to remove the constraints
// for the project detail view, allowing it to control its own full-width layout.
.content-wrapper--full-width {
  max-width: none;
  padding: 0;
  margin: 0;
}

*/
