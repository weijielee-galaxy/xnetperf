ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20042  10.155.84.4 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20043  10.155.84.4 &
ssh css-g88-002 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20044  10.155.84.4 &
ssh css-g88-002 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20045  10.155.84.4 &
ssh css-g88-003 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20046  10.155.84.4 &
ssh css-g88-003 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20047  10.155.84.4 &
