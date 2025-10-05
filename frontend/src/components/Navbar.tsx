import React from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

const Navbar: React.FC = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const isActive = (path: string) => {
    return location.pathname === path;
  };

  const navLinkClass = (path: string) => {
    const baseClass = 'px-3 py-2 rounded-md text-sm font-medium transition-colors';
    return isActive(path)
      ? `${baseClass} bg-indigo-700 text-white`
      : `${baseClass} text-indigo-100 hover:bg-indigo-500 hover:text-white`;
  };

  return (
    <nav className="bg-indigo-600 shadow-lg">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <h1
                className="text-white text-xl font-bold cursor-pointer"
                onClick={() => navigate('/dashboard')}
              >
                LogBuilder
              </h1>
            </div>
            <div className="ml-10 flex items-baseline space-x-4">
              <button
                onClick={() => navigate('/dashboard')}
                className={navLinkClass('/dashboard')}
              >
                Dashboard
              </button>
              <button
                onClick={() => navigate('/logs')}
                className={navLinkClass('/logs')}
              >
                Query
              </button>
              <button
                onClick={() => navigate('/explorer')}
                className={navLinkClass('/explorer')}
              >
                Explorer
              </button>
              <button
                onClick={() => navigate('/live')}
                className={navLinkClass('/live')}
              >
                Live Logs
              </button>
              <button
                onClick={() => navigate('/api-keys')}
                className={navLinkClass('/api-keys')}
              >
                API Keys
              </button>
            </div>
          </div>
          <div className="flex items-center">
            <span className="text-white mr-4">Welcome, {user?.username}</span>
            <button
              onClick={handleLogout}
              className="bg-indigo-500 hover:bg-indigo-700 text-white font-bold py-2 px-4 rounded transition-colors"
            >
              Logout
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navbar;
