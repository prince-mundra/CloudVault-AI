import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { AppShell } from '@/components/layout/AppShell';
import { ProtectedRoute } from './ProtectedRoute';
import { LandingPage } from '@/pages/LandingPage';
import { LoginPage } from '@/pages/LoginPage';
import { RegisterPage } from '@/pages/RegisterPage';
import { DashboardPage } from '@/pages/DashboardPage';
import { FilesPage } from '@/pages/FilesPage';
import { IntelligencePage } from '@/pages/IntelligencePage';
import { AssistantPage } from '@/pages/AssistantPage';
import { SearchPage } from '@/pages/SearchPage';
import { AnalyticsPage } from '@/pages/AnalyticsPage';
import { SharedPage } from '@/pages/SharedPage';
import { SettingsPage } from '@/pages/SettingsPage';

export const AppRoutes: React.FC = () => {
  return (
    <Routes>
      {/* Public Pages */}
      <Route path="/" element={<LandingPage />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />

      {/* Authenticated Workspace Pages (guarded by ProtectedRoute and wrapped in AppShell) */}
      <Route element={<ProtectedRoute />}>
        <Route element={<AppShell />}>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/files" element={<FilesPage />} />
          <Route path="/files/:folderId" element={<FilesPage />} />
          <Route path="/intelligence" element={<IntelligencePage />} />
          <Route path="/intelligence/:fileId" element={<IntelligencePage />} />
          <Route path="/assistant" element={<AssistantPage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route path="/analytics" element={<AnalyticsPage />} />
          <Route path="/shared" element={<SharedPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>
      </Route>

      {/* Catch-all Fallback */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
};
