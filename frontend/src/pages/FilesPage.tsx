import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Folder,
  FileText,
  Upload,
  Plus,
  Filter,
  MoreVertical,
  Download,
  Trash2,
  Edit2,
  Sparkles,
  ChevronRight,
  HardDrive,
  FileSpreadsheet,
  Image as ImageIcon,
  FileCode,
  Search,
  CheckCircle2,
  AlertCircle,
  Loader2,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu';
import { useNavigate } from 'react-router-dom';
import { api, FileItem, FolderItem } from '@/lib/api';
import { formatBytes } from '@/lib/utils';

function getFileIcon(filename: string, contentType?: string) {
  const ext = filename.split('.').pop()?.toLowerCase() || '';
  if (ext === 'pdf' || contentType?.includes('pdf')) {
    return <FileText className="w-5 h-5 text-[#7667F5]" />;
  }
  if (['xls', 'xlsx', 'csv'].includes(ext) || contentType?.includes('spreadsheet') || contentType?.includes('excel')) {
    return <FileSpreadsheet className="w-5 h-5 text-[#22C55E]" />;
  }
  if (['png', 'jpg', 'jpeg', 'webp', 'svg', 'gif'].includes(ext) || contentType?.includes('image')) {
    return <ImageIcon className="w-5 h-5 text-[#EAB308]" />;
  }
  if (['js', 'ts', 'jsx', 'tsx', 'py', 'go', 'json', 'html', 'css'].includes(ext)) {
    return <FileCode className="w-5 h-5 text-[#36C8D8]" />;
  }
  return <FileText className="w-5 h-5 text-[#9296A3]" />;
}

interface BreadcrumbItem {
  id: string;
  name: string;
}

export const FilesPage: React.FC = () => {
  const navigate = useNavigate();

  // Navigation & Location
  const [currentFolderId, setCurrentFolderId] = useState<string | null>(null);
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([]);

  // Data
  const [files, setFiles] = useState<FileItem[]>([]);
  const [folders, setFolders] = useState<FolderItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Search & Filter
  const [searchQuery, setSearchQuery] = useState('');
  const [filterType, setFilterType] = useState<string>('all');

  // Upload Modal State
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  const [uploadFileObj, setUploadFileObj] = useState<File | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [uploadSuccess, setUploadSuccess] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // New Folder Modal State
  const [isNewFolderOpen, setIsNewFolderOpen] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  const [creatingFolder, setCreatingFolder] = useState(false);
  const [folderError, setFolderError] = useState<string | null>(null);

  // Rename File Modal State
  const [fileToRename, setFileToRename] = useState<FileItem | null>(null);
  const [newFileName, setNewFileName] = useState('');
  const [renaming, setRenaming] = useState(false);
  const [renameError, setRenameError] = useState<string | null>(null);

  // Delete Confirm Modal State
  const [itemToDelete, setItemToDelete] = useState<{ type: 'file' | 'folder'; id: string; name: string } | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  // Fetch folders and files
  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [foldersRes, filesRes] = await Promise.all([
        api.listFolders(currentFolderId),
        api.listFiles({ folder_id: currentFolderId || 'root' }),
      ]);
      setFolders(foldersRes.data || []);
      setFiles(filesRes.data || []);
    } catch (err: any) {
      console.error('Failed to load storage items:', err);
      setError(err?.message || 'Failed to load files and folders');
    } finally {
      setLoading(false);
    }
  }, [currentFolderId]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Navigate folder
  const handleOpenFolder = (folder: FolderItem) => {
    setCurrentFolderId(folder.id);
    setBreadcrumbs((prev) => [...prev, { id: folder.id, name: folder.name }]);
  };

  const handleNavigateRoot = () => {
    setCurrentFolderId(null);
    setBreadcrumbs([]);
  };

  const handleNavigateBreadcrumb = (index: number) => {
    const target = breadcrumbs[index];
    setCurrentFolderId(target.id);
    setBreadcrumbs((prev) => prev.slice(0, index + 1));
  };

  // Upload handlers
  const handleSelectFile = (file: File) => {
    setUploadFileObj(file);
    setUploadProgress(0);
    setUploadError(null);
    setUploadSuccess(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleSelectFile(e.dataTransfer.files[0]);
    }
  };

  const handleStartUpload = async () => {
    if (!uploadFileObj) return;
    setUploading(true);
    setUploadError(null);
    setUploadProgress(0);

    try {
      await api.uploadFile(uploadFileObj, currentFolderId, (percent) => {
        setUploadProgress(percent);
      });
      setUploadSuccess(true);
      window.dispatchEvent(new Event('cloudvault-storage-update'));
      await loadData();
      setTimeout(() => {
        setIsUploadOpen(false);
        setUploadFileObj(null);
        setUploadProgress(0);
        setUploadSuccess(false);
        setUploading(false);
      }, 900);
    } catch (err: any) {
      console.error('Upload failed:', err);
      setUploadError(err?.message || 'Upload failed');
      setUploading(false);
    }
  };

  // Create folder handler
  const handleCreateFolder = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newFolderName.trim()) return;
    setCreatingFolder(true);
    setFolderError(null);

    try {
      await api.createFolder(newFolderName.trim(), currentFolderId);
      setIsNewFolderOpen(false);
      setNewFolderName('');
      await loadData();
    } catch (err: any) {
      console.error('Create folder failed:', err);
      setFolderError(err?.message || 'Failed to create folder');
    } finally {
      setCreatingFolder(false);
    }
  };

  // Rename file handler
  const handleRenameFile = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!fileToRename || !newFileName.trim()) return;
    setRenaming(true);
    setRenameError(null);

    try {
      await api.renameFile(fileToRename.id, newFileName.trim());
      setFileToRename(null);
      setNewFileName('');
      await loadData();
    } catch (err: any) {
      console.error('Rename file failed:', err);
      setRenameError(err?.message || 'Failed to rename file');
    } finally {
      setRenaming(false);
    }
  };

  // Delete item handler
  const handleDeleteItem = async () => {
    if (!itemToDelete) return;
    setDeleting(true);
    setDeleteError(null);

    try {
      if (itemToDelete.type === 'file') {
        await api.deleteFile(itemToDelete.id);
        window.dispatchEvent(new Event('cloudvault-storage-update'));
      } else {
        await api.deleteFolder(itemToDelete.id);
      }
      setItemToDelete(null);
      await loadData();
    } catch (err: any) {
      console.error('Delete failed:', err);
      setDeleteError(err?.message || 'Failed to delete item');
    } finally {
      setDeleting(false);
    }
  };

  // Download file handler
  const handleDownload = async (file: FileItem) => {
    try {
      await api.downloadFile(file.id, file.name);
    } catch (err: any) {
      console.error('Download failed:', err);
      alert(err?.message || 'Download failed');
    }
  };

  // Filter & Search files
  const filteredFiles = files.filter((f) => {
    const matchesSearch = f.name.toLowerCase().includes(searchQuery.toLowerCase());
    if (!matchesSearch) return false;

    if (filterType === 'all') return true;
    const ext = f.name.split('.').pop()?.toLowerCase() || '';
    if (filterType === 'pdf') return ext === 'pdf' || f.content_type?.includes('pdf');
    if (filterType === 'spreadsheet') return ['xls', 'xlsx', 'csv'].includes(ext) || f.content_type?.includes('excel');
    if (filterType === 'image') return ['png', 'jpg', 'jpeg', 'webp', 'svg', 'gif'].includes(ext) || f.content_type?.includes('image');
    if (filterType === 'code') return ['js', 'ts', 'jsx', 'tsx', 'py', 'go', 'json', 'html', 'css'].includes(ext);
    return true;
  });

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Action Toolbar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          {/* Breadcrumbs */}
          <nav className="flex items-center gap-1.5 text-xs text-[#9296A3] mb-1.5 flex-wrap">
            <button
              onClick={handleNavigateRoot}
              className={`hover:text-[#F3F4F6] transition-colors ${
                breadcrumbs.length === 0 ? 'text-[#F3F4F6] font-medium' : ''
              }`}
            >
              My Files
            </button>
            {breadcrumbs.map((crumb, idx) => (
              <React.Fragment key={crumb.id}>
                <ChevronRight className="w-3.5 h-3.5 text-[#9296A3]" />
                <button
                  onClick={() => handleNavigateBreadcrumb(idx)}
                  className={`hover:text-[#F3F4F6] transition-colors ${
                    idx === breadcrumbs.length - 1 ? 'text-[#F3F4F6] font-medium' : ''
                  }`}
                >
                  {crumb.name}
                </button>
              </React.Fragment>
            ))}
          </nav>
          <h1 className="text-2xl font-semibold tracking-tight text-[#F3F4F6]">
            {breadcrumbs.length > 0 ? breadcrumbs[breadcrumbs.length - 1].name : 'All Files'}
          </h1>
        </div>

        <div className="flex items-center gap-3 flex-wrap">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => {
              setNewFolderName('');
              setFolderError(null);
              setIsNewFolderOpen(true);
            }}
            className="flex items-center gap-1.5 text-xs"
          >
            <Plus className="w-3.5 h-3.5 text-[#9296A3]" /> New Folder
          </Button>

          <Button
            size="sm"
            onClick={() => {
              setUploadFileObj(null);
              setUploadProgress(0);
              setUploadError(null);
              setUploadSuccess(false);
              setIsUploadOpen(true);
            }}
            className="bg-[#7667F5] hover:bg-[#8679F7] text-white flex items-center gap-1.5 text-xs"
          >
            <Upload className="w-3.5 h-3.5" /> Upload File
          </Button>
        </div>
      </div>

      {/* Search and Filters Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-[#242731]">
        <div className="relative w-full sm:w-72">
          <Search className="w-4 h-4 text-[#9296A3] absolute left-3 top-1/2 -translate-y-1/2" />
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search files by name..."
            className="pl-9 h-9 text-xs bg-[#15171D] border-[#242731]"
          />
        </div>

        <div className="flex items-center gap-2 overflow-x-auto pb-1 sm:pb-0">
          <span className="text-xs text-[#9296A3] mr-1 flex items-center gap-1">
            <Filter className="w-3 h-3" /> Filter:
          </span>
          {['all', 'pdf', 'spreadsheet', 'image', 'code'].map((t) => (
            <button
              key={t}
              onClick={() => setFilterType(t)}
              className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors capitalize ${
                filterType === t
                  ? 'bg-[#7667F5] text-white'
                  : 'bg-[#15171D] border border-[#242731] text-[#9296A3] hover:text-[#F3F4F6] hover:bg-[#1F222B]'
              }`}
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div className="p-4 rounded-lg bg-[#EF4444]/10 border border-[#EF4444]/20 text-[#EF4444] text-xs flex items-center gap-2">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {loading ? (
        <div className="py-16 text-center text-sm text-[#9296A3] flex flex-col items-center justify-center space-y-3">
          <Loader2 className="w-6 h-6 animate-spin text-[#7667F5]" />
          <span>Loading vault contents...</span>
        </div>
      ) : (
        <>
          {/* Folders Section */}
          {folders.length > 0 && (
            <div>
              <h2 className="text-sm font-semibold text-[#F3F4F6] mb-3">Folders ({folders.length})</h2>
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
                {folders.map((folder) => (
                  <Card
                    key={folder.id}
                    onClick={() => handleOpenFolder(folder)}
                    className="p-4 hover:border-[#323644] transition-colors cursor-pointer group relative"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3 min-w-0">
                        <div className="w-9 h-9 rounded-lg bg-[#242731] flex items-center justify-center text-[#7667F5] group-hover:bg-[#7667F5]/10 transition-colors shrink-0">
                          <Folder className="w-5 h-5 fill-[#7667F5]/20 text-[#7667F5]" />
                        </div>
                        <div className="min-w-0">
                          <p className="text-sm font-medium text-[#F3F4F6] truncate group-hover:text-[#7667F5] transition-colors">
                            {folder.name}
                          </p>
                          <p className="text-xs text-[#9296A3]">Folder</p>
                        </div>
                      </div>

                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <button
                            onClick={(e) => e.stopPropagation()}
                            className="p-1 rounded hover:bg-[#242731] text-[#9296A3] hover:text-[#F3F4F6] transition-colors"
                          >
                            <MoreVertical className="w-4 h-4" />
                          </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            danger
                            onClick={(e) => {
                              e.stopPropagation();
                              setDeleteError(null);
                              setItemToDelete({ type: 'folder', id: folder.id, name: folder.name });
                            }}
                          >
                            <Trash2 className="w-3.5 h-3.5 mr-2" /> Delete Folder
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          )}

          {/* Files Section */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-semibold text-[#F3F4F6]">
                Files ({filteredFiles.length}{filteredFiles.length !== files.length ? ` of ${files.length}` : ''})
              </h2>
            </div>

            {filteredFiles.length > 0 ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {filteredFiles.map((file) => {
                  const isPdf = file.name.toLowerCase().endsWith('.pdf') || file.content_type?.includes('pdf');
                  return (
                    <Card
                      key={file.id}
                      className="p-4 hover:border-[#323644] transition-colors flex flex-col justify-between h-40 group"
                    >
                      <div className="flex items-start justify-between">
                        <div
                          onClick={() => {
                            if (isPdf) {
                              navigate(`/intelligence/${file.id}`);
                            }
                          }}
                          className={`flex items-center gap-3 min-w-0 ${isPdf ? 'cursor-pointer' : ''}`}
                        >
                          <div className="w-9 h-9 rounded-lg bg-[#7667F5]/10 border border-[#7667F5]/20 flex items-center justify-center text-[#7667F5] shrink-0">
                            {getFileIcon(file.name, file.content_type)}
                          </div>
                          <div className="min-w-0">
                            <p
                              className={`text-sm font-medium text-[#F3F4F6] truncate ${
                                isPdf ? 'group-hover:text-[#7667F5] transition-colors' : ''
                              }`}
                              title={file.name}
                            >
                              {file.name}
                            </p>
                            <p className="text-xs text-[#9296A3]">{formatBytes(file.size)}</p>
                          </div>
                        </div>

                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <button className="p-1 rounded hover:bg-[#242731] text-[#9296A3] hover:text-[#F3F4F6] transition-colors">
                              <MoreVertical className="w-4 h-4" />
                            </button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => handleDownload(file)}>
                              <Download className="w-3.5 h-3.5 mr-2 text-[#7667F5]" /> Download
                            </DropdownMenuItem>

                            {isPdf && (
                              <DropdownMenuItem onClick={() => navigate(`/intelligence/${file.id}`)}>
                                <Sparkles className="w-3.5 h-3.5 mr-2 text-[#36C8D8]" /> AI Intelligence
                              </DropdownMenuItem>
                            )}

                            <DropdownMenuItem
                              onClick={() => {
                                setFileToRename(file);
                                setNewFileName(file.name);
                                setRenameError(null);
                              }}
                            >
                              <Edit2 className="w-3.5 h-3.5 mr-2 text-[#9296A3]" /> Rename
                            </DropdownMenuItem>

                            <DropdownMenuSeparator />

                            <DropdownMenuItem
                              danger
                              onClick={() => {
                                setDeleteError(null);
                                setItemToDelete({ type: 'file', id: file.id, name: file.name });
                              }}
                            >
                              <Trash2 className="w-3.5 h-3.5 mr-2" /> Delete
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>

                      <div className="flex items-center justify-between pt-3 border-t border-[#242731] text-xs text-[#9296A3]">
                        <span>{new Date(file.created_at).toLocaleDateString()}</span>
                        <div className="flex items-center gap-2">
                          <span
                            className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${
                              isPdf
                                ? 'bg-[#36C8D8]/10 text-[#36C8D8] border-[#36C8D8]/20'
                                : 'bg-[#242731] text-[#9296A3] border-[#323644]'
                            }`}
                          >
                            {isPdf ? 'Indexed' : 'Encrypted'}
                          </span>
                        </div>
                      </div>
                    </Card>
                  );
                })}
              </div>
            ) : (
              <div className="py-12 border border-dashed border-[#242731] rounded-xl flex flex-col items-center justify-center text-center space-y-3">
                <div className="w-12 h-12 rounded-full bg-[#1F222B] border border-[#242731] flex items-center justify-center text-[#9296A3]">
                  <HardDrive className="w-6 h-6" />
                </div>
                <div>
                  <p className="text-sm font-medium text-[#F3F4F6]">
                    {searchQuery ? 'No matching files found' : 'No files in this location'}
                  </p>
                  <p className="text-xs text-[#9296A3] mt-1 max-w-sm">
                    {searchQuery
                      ? 'Try adjusting your search query or clear the filter.'
                      : 'Upload files to store them securely with AES-256-GCM encryption.'}
                  </p>
                </div>
                {!searchQuery && (
                  <Button
                    size="sm"
                    onClick={() => {
                      setUploadFileObj(null);
                      setUploadProgress(0);
                      setUploadError(null);
                      setUploadSuccess(false);
                      setIsUploadOpen(true);
                    }}
                    className="bg-[#7667F5] hover:bg-[#8679F7] text-white text-xs mt-2"
                  >
                    <Upload className="w-3.5 h-3.5 mr-1.5" /> Upload File
                  </Button>
                )}
              </div>
            )}
          </div>
        </>
      )}

      {/* Upload File Dialog */}
      <Dialog open={isUploadOpen} onOpenChange={setIsUploadOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-[#F3F4F6]">
              Upload to {breadcrumbs.length > 0 ? breadcrumbs[breadcrumbs.length - 1].name : 'Root Vault'}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2">
            {/* Drag & Drop Zone */}
            <div
              onDragOver={(e) => e.preventDefault()}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
              className="border-2 border-dashed border-[#242731] hover:border-[#7667F5]/50 bg-[#15171D] hover:bg-[#1A1D25] rounded-xl p-6 text-center cursor-pointer transition-colors flex flex-col items-center justify-center space-y-2"
            >
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                onChange={(e) => {
                  if (e.target.files && e.target.files[0]) {
                    handleSelectFile(e.target.files[0]);
                  }
                }}
              />
              <div className="w-10 h-10 rounded-full bg-[#7667F5]/10 border border-[#7667F5]/20 flex items-center justify-center text-[#7667F5]">
                <Upload className="w-5 h-5" />
              </div>
              <p className="text-xs font-medium text-[#F3F4F6]">
                Click or drag & drop file to upload
              </p>
              <p className="text-[11px] text-[#9296A3]">
                Supports PDF, spreadsheets, images, text, code (up to 50 MB)
              </p>
            </div>

            {/* Selected File Details */}
            {uploadFileObj && (
              <div className="p-3 rounded-lg bg-[#15171D] border border-[#242731] flex items-center justify-between text-xs">
                <div className="flex items-center gap-2.5 min-w-0">
                  {getFileIcon(uploadFileObj.name, uploadFileObj.type)}
                  <div className="min-w-0">
                    <p className="font-medium text-[#F3F4F6] truncate max-w-[240px]">
                      {uploadFileObj.name}
                    </p>
                    <p className="text-[#9296A3] text-[11px]">{formatBytes(uploadFileObj.size)}</p>
                  </div>
                </div>
                {uploadSuccess && <CheckCircle2 className="w-4 h-4 text-[#22C55E] shrink-0" />}
              </div>
            )}

            {/* Upload Progress Bar */}
            {uploading && (
              <div className="space-y-1.5">
                <div className="flex justify-between text-xs text-[#9296A3]">
                  <span>Uploading & encrypting...</span>
                  <span className="font-medium text-[#F3F4F6]">{uploadProgress}%</span>
                </div>
                <Progress value={uploadProgress} className="h-1.5" />
              </div>
            )}

            {uploadError && (
              <div className="p-3 rounded-lg bg-[#EF4444]/10 border border-[#EF4444]/20 text-[#EF4444] text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{uploadError}</span>
              </div>
            )}

            {uploadSuccess && (
              <div className="p-3 rounded-lg bg-[#22C55E]/10 border border-[#22C55E]/20 text-[#22C55E] text-xs flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4 shrink-0" />
                <span>Upload complete! Stored with AES-256-GCM.</span>
              </div>
            )}
          </div>

          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setIsUploadOpen(false)}
              disabled={uploading}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={handleStartUpload}
              disabled={!uploadFileObj || uploading || uploadSuccess}
              className="bg-[#7667F5] hover:bg-[#8679F7] text-white"
            >
              {uploading ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 animate-spin mr-1.5" /> Uploading...
                </>
              ) : (
                'Upload'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* New Folder Dialog */}
      <Dialog open={isNewFolderOpen} onOpenChange={setIsNewFolderOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-[#F3F4F6]">Create New Folder</DialogTitle>
          </DialogHeader>

          <form onSubmit={handleCreateFolder} className="space-y-4 py-2">
            <div className="space-y-1.5">
              <label className="text-xs text-[#9296A3] font-medium">Folder Name</label>
              <Input
                value={newFolderName}
                onChange={(e) => setNewFolderName(e.target.value)}
                placeholder="e.g. Q4 Financials"
                autoFocus
                className="h-9 text-xs bg-[#15171D] border-[#242731]"
              />
            </div>

            {folderError && (
              <div className="p-2.5 rounded-lg bg-[#EF4444]/10 border border-[#EF4444]/20 text-[#EF4444] text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{folderError}</span>
              </div>
            )}

            <DialogFooter className="gap-2 sm:gap-0 pt-2">
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={() => setIsNewFolderOpen(false)}
                disabled={creatingFolder}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={!newFolderName.trim() || creatingFolder}
                className="bg-[#7667F5] hover:bg-[#8679F7] text-white"
              >
                {creatingFolder ? 'Creating...' : 'Create Folder'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Rename File Dialog */}
      <Dialog open={!!fileToRename} onOpenChange={(open) => !open && setFileToRename(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-[#F3F4F6]">Rename File</DialogTitle>
          </DialogHeader>

          <form onSubmit={handleRenameFile} className="space-y-4 py-2">
            <div className="space-y-1.5">
              <label className="text-xs text-[#9296A3] font-medium">New Name</label>
              <Input
                value={newFileName}
                onChange={(e) => setNewFileName(e.target.value)}
                autoFocus
                className="h-9 text-xs bg-[#15171D] border-[#242731]"
              />
            </div>

            {renameError && (
              <div className="p-2.5 rounded-lg bg-[#EF4444]/10 border border-[#EF4444]/20 text-[#EF4444] text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{renameError}</span>
              </div>
            )}

            <DialogFooter className="gap-2 sm:gap-0 pt-2">
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={() => setFileToRename(null)}
                disabled={renaming}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={!newFileName.trim() || renaming}
                className="bg-[#7667F5] hover:bg-[#8679F7] text-white"
              >
                {renaming ? 'Saving...' : 'Rename'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Confirm Delete Dialog */}
      <Dialog open={!!itemToDelete} onOpenChange={(open) => !open && setItemToDelete(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-[#F3F4F6]">
              Delete {itemToDelete?.type === 'folder' ? 'Folder' : 'File'}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-3 py-2 text-xs text-[#9296A3]">
            <p>
              Are you sure you want to delete{' '}
              <strong className="text-[#F3F4F6]">{itemToDelete?.name}</strong>?
            </p>
            {itemToDelete?.type === 'folder' ? (
              <p>The folder must be empty to be deleted.</p>
            ) : (
              <p>This action will release storage quota and remove the encrypted file.</p>
            )}

            {deleteError && (
              <div className="p-2.5 rounded-lg bg-[#EF4444]/10 border border-[#EF4444]/20 text-[#EF4444] text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{deleteError}</span>
              </div>
            )}
          </div>

          <DialogFooter className="gap-2 sm:gap-0 pt-2">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={() => setItemToDelete(null)}
              disabled={deleting}
            >
              Cancel
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={handleDeleteItem}
              disabled={deleting}
              className="bg-[#EF4444] hover:bg-[#DC2626] text-white"
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};