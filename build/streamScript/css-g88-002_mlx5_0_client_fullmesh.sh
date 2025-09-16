ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20012  10.155.84.2 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20013  10.155.84.2 &
ssh css-g88-003 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20014  10.155.84.2 &
ssh css-g88-003 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20015  10.155.84.2 &
ssh css-g88-004 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20016  10.155.84.2 &
ssh css-g88-004 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20017  10.155.84.2 &
